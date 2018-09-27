package publicapi

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"time"
	)

type Config interface {
	SendTransactionTimeout() time.Duration
	GetTransactionStatusGrace() time.Duration
}

type service struct {
	config          Config
	transactionPool services.TransactionPool
	virtualMachine  services.VirtualMachine
	reporting       log.BasicLogger

	txWaiter *txWaiter
}

func NewPublicApi(
	ctx context.Context,
	config Config,
	transactionPool services.TransactionPool,
	virtualMachine services.VirtualMachine,
	reporting log.BasicLogger,
) services.PublicApi {
	s := &service{
		config:          config,
		transactionPool: transactionPool,
		virtualMachine:  virtualMachine,
		reporting:       reporting.For(log.Service("public-api")),

		txWaiter: newTxWaiter(ctx),
	}

	transactionPool.RegisterTransactionResultsHandler(s)

	return s
}

func (s *service) HandleTransactionResults(input *handlers.HandleTransactionResultsInput) (*handlers.HandleTransactionResultsOutput, error) {
	for _, txReceipt := range input.TransactionReceipts {
		s.reporting.Info("transaction reported as committed", log.String("flow", "checkpoint"), log.Stringable("txHash", txReceipt.Txhash()))
		s.txWaiter.reportCompleted(txReceipt, input.BlockHeight, input.Timestamp)
	}
	return &handlers.HandleTransactionResultsOutput{}, nil
}

func (s *service) HandleTransactionError(input *handlers.HandleTransactionErrorInput) (*handlers.HandleTransactionErrorOutput, error) {
	//TODO implement
	s.reporting.Info("transaction reported as errored", log.String("flow", "checkpoint"), log.Stringable("txHash", input.Txhash), log.Stringable("tx-status", input.TransactionStatus))

	return &handlers.HandleTransactionErrorOutput{}, nil
}

func (s *service) SendTransaction(input *services.SendTransactionInput) (*services.SendTransactionOutput, error) {
	tx := input.ClientRequest.SignedTransaction()
	txHash := digest.CalcTxHash(input.ClientRequest.SignedTransaction().Transaction())

	s.reporting.Info("transaction received via public api", log.String("flow", "checkpoint"), log.Stringable("txHash", txHash))

	waitContext := s.txWaiter.createTxWaitCtx(txHash)
	defer waitContext.cleanup()

	txResponse, err := s.transactionPool.AddNewTransaction(&services.AddNewTransactionInput{
		SignedTransaction: tx,
	})

	if err != nil {
		s.reporting.Info("adding transaction to TransactionPool failed", log.Error(err), log.String("flow", "checkpoint"), log.Stringable("txHash", txHash))
		return prepareResponse(txResponse), errors.Errorf("error '%s' for transaction result", txResponse)
	}
	if txResponse.TransactionStatus == protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED {
		return prepareResponse(txResponse), nil
	}

	ta, err := waitContext.until(s.config.SendTransactionTimeout())
	if err != nil {
		s.reporting.Info("waiting for transaction to be processed failed", log.Error(err), log.String("flow", "checkpoint"), log.Stringable("txHash", txHash))
		return prepareResponse(txResponse), err
	}
	return prepareResponse(ta), nil
}

func prepareResponse(transactionOutput *services.AddNewTransactionOutput) *services.SendTransactionOutput {
	var receiptForClient *protocol.TransactionReceiptBuilder = nil

	if receipt := transactionOutput.TransactionReceipt; receipt != nil {
		receiptForClient = &protocol.TransactionReceiptBuilder{
			Txhash:              receipt.Txhash(),
			ExecutionResult:     receipt.ExecutionResult(),
			OutputArgumentArray: receipt.OutputArgumentArray(),
		}
	}

	response := &client.SendTransactionResponseBuilder{
		RequestStatus:      translateTxStatusToResponseCode(transactionOutput.TransactionStatus),
		TransactionReceipt: receiptForClient,
		TransactionStatus:  transactionOutput.TransactionStatus,
		BlockHeight:        transactionOutput.BlockHeight,
		BlockTimestamp:     transactionOutput.BlockTimestamp,
	}

	return &services.SendTransactionOutput{ClientResponse: response.Build()}
}

func translateTxStatusToResponseCode(txStatus protocol.TransactionStatus) protocol.RequestStatus {
	switch txStatus {
	case protocol.TRANSACTION_STATUS_COMMITTED:
		return protocol.REQUEST_STATUS_COMPLETED
	case protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED:
		return protocol.REQUEST_STATUS_COMPLETED
	case protocol.TRANSACTION_STATUS_PENDING:
		return protocol.REQUEST_STATUS_IN_PROCESS
	case protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING:
		return protocol.REQUEST_STATUS_IN_PROCESS
	case protocol.TRANSACTION_STATUS_NO_RECORD_FOUND:
		return protocol.REQUEST_STATUS_NOT_FOUND
	case protocol.TRANSACTION_STATUS_REJECTED_UNSUPPORTED_VERSION:
		return protocol.REQUEST_STATUS_REJECTED
	case protocol.TRANSACTION_STATUS_REJECTED_VIRTUAL_CHAIN_MISMATCH:
		return protocol.REQUEST_STATUS_REJECTED
	case protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_WINDOW_EXCEEDED:
		return protocol.REQUEST_STATUS_REJECTED
	case protocol.TRANSACTION_STATUS_REJECTED_SIGNATURE_MISMATCH:
		return protocol.REQUEST_STATUS_REJECTED
	case protocol.TRANSACTION_STATUS_REJECTED_UNKNOWN_SIGNER_SCHEME:
		return protocol.REQUEST_STATUS_REJECTED
	case protocol.TRANSACTION_STATUS_REJECTED_GLOBAL_PRE_ORDER:
		return protocol.REQUEST_STATUS_REJECTED
	case protocol.TRANSACTION_STATUS_REJECTED_VIRTUAL_CHAIN_PRE_ORDER:
		return protocol.REQUEST_STATUS_REJECTED
	case protocol.TRANSACTION_STATUS_REJECTED_SMART_CONTRACT_PRE_ORDER:
		return protocol.REQUEST_STATUS_REJECTED
	case protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_PRECEDES_NODE_TIME:
		return protocol.REQUEST_STATUS_REJECTED
	case protocol.TRANSACTION_STATUS_REJECTED_CONGESTION:
		return protocol.REQUEST_STATUS_CONGESTION
	}
	return protocol.REQUEST_STATUS_RESERVED
}

func (s *service) CallMethod(input *services.CallMethodInput) (*services.CallMethodOutput, error) {
	s.reporting.Info("enter CallMethod")
	defer s.reporting.Info("exit CallMethod")

	output, err := s.virtualMachine.RunLocalMethod(&services.RunLocalMethodInput{
		Transaction: input.ClientRequest.Transaction(),
	})
	if err != nil {
		s.reporting.Info("running local method on VirtualMachine failed", log.Error(err))
		return nil, err
	}
	return &services.CallMethodOutput{
		ClientResponse: (&client.CallMethodResponseBuilder{
			// TODO need to fill up this struct
			RequestStatus:       protocol.REQUEST_STATUS_COMPLETED,
			OutputArgumentArray: output.OutputArgumentArray,
		}).Build(),
	}, nil
}

func (s *service) GetTransactionStatus(input *services.GetTransactionStatusInput) (*services.GetTransactionStatusOutput, error) {
	panic("Not implemented")
}

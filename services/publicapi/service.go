package publicapi

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"time"
)

var LogTag = log.Service("public-api")

type Config interface {
	SendTransactionTimeout() time.Duration
	VirtualChainId() primitives.VirtualChainId
}

type txResponse struct {
	transactionStatus  protocol.TransactionStatus
	transactionReceipt *protocol.TransactionReceipt
	blockHeight        primitives.BlockHeight
	blockTimestamp     primitives.TimestampNano
}

type service struct {
	ctx             context.Context
	config          Config
	transactionPool services.TransactionPool
	virtualMachine  services.VirtualMachine
	blockStorage    services.BlockStorage
	logger          log.BasicLogger

	waiter *waiter
}

func NewPublicApi(
	ctx context.Context,
	config Config,
	transactionPool services.TransactionPool,
	virtualMachine services.VirtualMachine,
	blockStorage services.BlockStorage,
	logger log.BasicLogger,
) services.PublicApi {
	s := &service{
		ctx:             ctx,
		config:          config,
		transactionPool: transactionPool,
		virtualMachine:  virtualMachine,
		blockStorage:    blockStorage,
		logger:          logger.WithTags(LogTag),

		waiter: newWaiter(ctx),
	}

	transactionPool.RegisterTransactionResultsHandler(s)

	return s
}

func (s *service) HandleTransactionResults(input *handlers.HandleTransactionResultsInput) (*handlers.HandleTransactionResultsOutput, error) {
	for _, txReceipt := range input.TransactionReceipts {
		s.logger.Info("transaction reported as committed", log.String("flow", "checkpoint"), log.Stringable("txHash", txReceipt.Txhash()))
		s.waiter.complete(txReceipt.Txhash().KeyForMap(),
			&waiterObject{&txResponse{
				transactionStatus:  protocol.TRANSACTION_STATUS_COMMITTED,
				transactionReceipt: txReceipt,
				blockHeight:        input.BlockHeight,
				blockTimestamp:     input.Timestamp,
			}})
	}
	return &handlers.HandleTransactionResultsOutput{}, nil
}

func (s *service) HandleTransactionError(input *handlers.HandleTransactionErrorInput) (*handlers.HandleTransactionErrorOutput, error) {
	s.logger.Info("transaction reported as errored", log.String("flow", "checkpoint"), log.Stringable("txHash", input.Txhash), log.Stringable("tx-status", input.TransactionStatus))
	s.waiter.complete(input.Txhash.KeyForMap(),
		&waiterObject{&txResponse{
			transactionStatus:  input.TransactionStatus,
			transactionReceipt: nil,
			blockHeight:        input.BlockHeight,
			blockTimestamp:     input.BlockTimestamp,
		}})
	return &handlers.HandleTransactionErrorOutput{}, nil
}

func (s *service) SendTransaction(input *services.SendTransactionInput) (*services.SendTransactionOutput, error) {
	if input.ClientRequest == nil {
		err := errors.Errorf("error missing input (client request is nil")
		s.logger.Info("send transaction received via public api", log.Error(err))
		return nil, err
	}

	tx := input.ClientRequest.SignedTransaction()
	if txStatus := isTransactionRequestValid(s.config, tx.Transaction().VirtualChainId()); txStatus != protocol.TRANSACTION_STATUS_RESERVED {
		return toSendTxOutput(&txResponse{transactionStatus: txStatus}), nil
	}

	txHash := digest.CalcTxHash(tx.Transaction())
	s.logger.Info("transaction received via public api", log.String("flow", "checkpoint"), log.Stringable("txHash", txHash))

	waitResult := s.waiter.add(txHash.KeyForMap())

	addResp, err := s.transactionPool.AddNewTransaction(&services.AddNewTransactionInput{SignedTransaction: tx})
	if err != nil {
		s.waiter.deleteByChannel(waitResult)
		s.logger.Info("adding transaction to TransactionPool failed", log.Error(err), log.String("flow", "checkpoint"), log.Stringable("txHash", txHash))
		return toSendTxOutput(toTxResponse(addResp)), errors.Errorf("error '%s' for transaction result", addResp)
	}

	if addResp.TransactionStatus == protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED {
		s.waiter.deleteByChannel(waitResult)
		return toSendTxOutput(toTxResponse(addResp)), nil
	}

	obj, err := s.waiter.wait(waitResult, s.config.SendTransactionTimeout())
	if err != nil {
		s.logger.Info("waiting for transaction to be processed failed", log.Error(err), log.String("flow", "checkpoint"), log.Stringable("txHash", txHash))
		return toSendTxOutput(toTxResponse(addResp)), err
	}
	return toSendTxOutput(obj.payload.(*txResponse)), nil
}

func toTxResponse(t *services.AddNewTransactionOutput) *txResponse {
	return &txResponse{
		transactionStatus:  t.TransactionStatus,
		transactionReceipt: t.TransactionReceipt,
		blockHeight:        t.BlockHeight,
		blockTimestamp:     t.BlockTimestamp,
	}
}

func toSendTxOutput(transactionOutput *txResponse) *services.SendTransactionOutput {
	var receiptForClient *protocol.TransactionReceiptBuilder = nil

	if receipt := transactionOutput.transactionReceipt; receipt != nil {
		receiptForClient = &protocol.TransactionReceiptBuilder{
			Txhash:              receipt.Txhash(),
			ExecutionResult:     receipt.ExecutionResult(),
			OutputArgumentArray: receipt.OutputArgumentArray(),
		}
	}

	response := &client.SendTransactionResponseBuilder{
		RequestStatus:      translateTxStatusToResponseCode(transactionOutput.transactionStatus),
		TransactionReceipt: receiptForClient,
		TransactionStatus:  transactionOutput.transactionStatus,
		BlockHeight:        transactionOutput.blockHeight,
		BlockTimestamp:     transactionOutput.blockTimestamp,
	}

	return &services.SendTransactionOutput{ClientResponse: response.Build()}
}

func (s *service) CallMethod(input *services.CallMethodInput) (*services.CallMethodOutput, error) {
	s.logger.Info("enter CallMethod")
	defer s.logger.Info("exit CallMethod")

	output, err := s.virtualMachine.RunLocalMethod(&services.RunLocalMethodInput{
		Transaction: input.ClientRequest.Transaction(),
	})
	if err != nil {
		s.logger.Info("running local method on VirtualMachine failed", log.Error(err))
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
	if input.ClientRequest == nil {
		err := errors.Errorf("error: missing input (client request is nil")
		s.logger.Info("get transaction status received via public api", log.Error(err))
		return nil, err
	}

	s.logger.Info("get transaction status request received via public api", log.String("flow", "checkpoint"), log.Stringable("txHash", input.ClientRequest.Txhash()))
	txReceipt, err := s.transactionPool.GetCommittedTransactionReceipt(&services.GetCommittedTransactionReceiptInput{
		Txhash:               input.ClientRequest.Txhash(),
		TransactionTimestamp: input.ClientRequest.TransactionTimestamp(),
	})
	if err != nil {
		s.logger.Info("get transaction status via public api failed in transactionPool", log.Error(err), log.String("flow", "checkpoint"), log.Stringable("txHash", input.ClientRequest.Txhash()))
		return toGetTxOutput(txStatusToTxResponse(txReceipt)), err
	}
	if txReceipt.TransactionStatus != protocol.TRANSACTION_STATUS_NO_RECORD_FOUND {
		return toGetTxOutput(txStatusToTxResponse(txReceipt)), nil
	}

	blockReceipt, err := s.blockStorage.GetTransactionReceipt(&services.GetTransactionReceiptInput{
		Txhash:               input.ClientRequest.Txhash(),
		TransactionTimestamp: input.ClientRequest.TransactionTimestamp(),
	})
	if err != nil {
		s.logger.Info("get transaction status via public api failed in blockStorage", log.Error(err), log.String("flow", "checkpoint"), log.Stringable("txHash", input.ClientRequest.Txhash()))
		return toGetTxOutput(blockToTxResponse(blockReceipt)), err
	}
	return toGetTxOutput(blockToTxResponse(blockReceipt)), nil
}

func txStatusToTxResponse(txStatus *services.GetCommittedTransactionReceiptOutput) *txResponse {
	return &txResponse{
		transactionStatus:  txStatus.TransactionStatus,
		transactionReceipt: txStatus.TransactionReceipt,
		blockHeight:        txStatus.BlockHeight,
		blockTimestamp:     txStatus.BlockTimestamp,
	}
}

func blockToTxResponse(bReceipt *services.GetTransactionReceiptOutput) *txResponse {
	status := protocol.TRANSACTION_STATUS_NO_RECORD_FOUND
	if bReceipt.TransactionReceipt != nil {
		status = protocol.TRANSACTION_STATUS_COMMITTED
	}
	return &txResponse{
		transactionStatus:  status,
		transactionReceipt: bReceipt.TransactionReceipt,
		blockHeight:        bReceipt.BlockHeight,
		blockTimestamp:     bReceipt.BlockTimestamp,
	}
}

func toGetTxOutput(transactionOutput *txResponse) *services.GetTransactionStatusOutput {
	var receiptForClient *protocol.TransactionReceiptBuilder = nil

	if receipt := transactionOutput.transactionReceipt; receipt != nil {
		receiptForClient = &protocol.TransactionReceiptBuilder{
			Txhash:              receipt.Txhash(),
			ExecutionResult:     receipt.ExecutionResult(),
			OutputArgumentArray: receipt.OutputArgumentArray(),
		}
	}

	response := &client.GetTransactionStatusResponseBuilder{
		RequestStatus:      translateTxStatusToResponseCode(transactionOutput.transactionStatus),
		TransactionReceipt: receiptForClient,
		TransactionStatus:  transactionOutput.transactionStatus,
		BlockHeight:        transactionOutput.blockHeight,
		BlockTimestamp:     transactionOutput.blockTimestamp,
	}

	return &services.GetTransactionStatusOutput{ClientResponse: response.Build()}
}

// General helpers
func isTransactionRequestValid(config Config, vcId primitives.VirtualChainId) protocol.TransactionStatus {
	if config.VirtualChainId() != vcId {
		return protocol.TRANSACTION_STATUS_REJECTED_VIRTUAL_CHAIN_MISMATCH
	}

	return protocol.TRANSACTION_STATUS_RESERVED // used as an OK
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
	case protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_AHEAD_OF_NODE_TIME:
		return protocol.REQUEST_STATUS_REJECTED
	case protocol.TRANSACTION_STATUS_REJECTED_CONGESTION:
		return protocol.REQUEST_STATUS_CONGESTION
	}
	return protocol.REQUEST_STATUS_RESERVED
}

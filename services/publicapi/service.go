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
	me := &service{
		config:          config,
		transactionPool: transactionPool,
		virtualMachine:  virtualMachine,
		reporting:       reporting.For(log.Service("public-api")),

		txWaiter: newWaiter(ctx),
	}

	transactionPool.RegisterTransactionResultsHandler(me)

	return me
}

func (s *service) HandleTransactionResults(input *handlers.HandleTransactionResultsInput) (*handlers.HandleTransactionResultsOutput, error) {
	for _, txReceipt := range input.TransactionReceipts {
		s.txWaiter.reportCompleted(txReceipt, input.BlockHeight, input.Timestamp)
	}
	return &handlers.HandleTransactionResultsOutput{}, nil
}

func (s *service) SendTransaction(input *services.SendTransactionInput) (*services.SendTransactionOutput, error) {
	s.reporting.Info("enter SendTransaction")
	defer s.reporting.Info("exit SendTransaction")

	tx := input.ClientRequest.SignedTransaction()
	txHash := digest.CalcTxHash(input.ClientRequest.SignedTransaction().Transaction())

	wait := s.txWaiter.prepareFor(txHash)
	defer wait.cleanup()

	txResponse, err := s.transactionPool.AddNewTransaction(&services.AddNewTransactionInput{
		SignedTransaction: tx,
	})

	if err != nil {
		return prepareResponse(txResponse), errors.Errorf("error '%s' for transaction result", txResponse)
	}
	if txResponse.TransactionStatus == protocol.TRANSACTION_STATUS_DUPLCIATE_TRANSACTION_ALREADY_COMMITTED {
		return prepareResponse(txResponse), nil
	}

	ta, err := wait.until(s.config.SendTransactionTimeout())
	// TODO return pending response on timeout error
	if err != nil {
		return nil, err
	}
	return prepareResponse(ta), nil
}

func prepareResponse(transactionOutput *services.AddNewTransactionOutput) *services.SendTransactionOutput {
	var receiptForClient *protocol.TransactionReceiptBuilder = nil

	if receipt := transactionOutput.TransactionReceipt; receipt != nil {
		mabs := make([]*protocol.MethodArgumentBuilder, 0, 1)
		// TODO replace with Tals implementation
		oai := receipt.OutputArgumentsIterator()
		for oai.HasNext() {
			ma := oai.NextOutputArguments()
			mabs = append(mabs, &protocol.MethodArgumentBuilder{
				Name:        ma.Name(),
				Type:        ma.Type(),
				Uint32Value: ma.Uint32Value(),
				Uint64Value: ma.Uint64Value(),
				StringValue: ma.StringValue(),
				BytesValue:  ma.BytesValue()},
			)
		}
		receiptForClient = &protocol.TransactionReceiptBuilder{
			Txhash:          receipt.Txhash(),
			ExecutionResult: receipt.ExecutionResult(),
			OutputArguments: mabs,
		}
	}

	response := &client.SendTransactionResponseBuilder{
		TransactionReceipt: receiptForClient,
		TransactionStatus:  transactionOutput.TransactionStatus,
		BlockHeight:        transactionOutput.BlockHeight,
		BlockTimestamp:     transactionOutput.BlockTimestamp,
	}

	return &services.SendTransactionOutput{ClientResponse: response.Build()}
}

func (s *service) CallMethod(input *services.CallMethodInput) (*services.CallMethodOutput, error) {
	s.reporting.Info("enter CallMethod")
	defer s.reporting.Info("exit CallMethod")
	// TODO get block height for input ?
	rlm, err := s.virtualMachine.RunLocalMethod(&services.RunLocalMethodInput{
		Transaction: input.ClientRequest.Transaction(),
	})
	if err != nil {
		return nil, err
	}
	var oa []*protocol.MethodArgumentBuilder
	for _, arg := range rlm.OutputArguments {
		switch arg.Type() {
		case protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE:
			oa = []*protocol.MethodArgumentBuilder{
				{Name: arg.Name(), Type: arg.Type(), Uint64Value: arg.Uint64Value()},
			}
		}
	}
	return &services.CallMethodOutput{
		ClientResponse: (&client.CallMethodResponseBuilder{
			OutputArguments: oa,
		}).Build(),
	}, nil
}

func (s *service) GetTransactionStatus(input *services.GetTransactionStatusInput) (*services.GetTransactionStatusOutput, error) {
	panic("Not implemented")
}

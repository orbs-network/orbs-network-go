package publicapi

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"sync"
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

	mutex  sync.RWMutex
	txChan map[string]chan *services.AddNewTransactionOutput
}

func NewPublicApi(
	config Config,
	transactionPool services.TransactionPool,
	virtualMachine services.VirtualMachine,
	reporting log.BasicLogger,
) services.PublicApi {

	return &service{
		config:          config,
		transactionPool: transactionPool,
		virtualMachine:  virtualMachine,
		reporting:       reporting.For(log.Service("public-api")),

		mutex:  sync.RWMutex{},
		txChan: map[string]chan *services.AddNewTransactionOutput{},
	}
}

func (s *service) SendTransaction(input *services.SendTransactionInput) (*services.SendTransactionOutput, error) {
	s.reporting.Info("enter SendTransaction")
	defer s.reporting.Info("exit SendTransaction")

	tx := input.ClientRequest.SignedTransaction()
	alreadyCommitted, err := s.transactionPool.AddNewTransaction(&services.AddNewTransactionInput{
		SignedTransaction: tx,
	})

	//TODO handle alreadyCommitted and errors
	if err != nil {
		return nil, errors.Wrapf(err, "failed queuing transaction")
	}
	if alreadyCommitted.TransactionStatus == protocol.TRANSACTION_STATUS_COMMITTED ||
		alreadyCommitted.TransactionStatus == protocol.TRANSACTION_STATUS_DUPLCIATE_TRANSACTION_ALREADY_COMMITTED {
		return prepareResponse(alreadyCommitted), nil
	}

	txId := digest.CalcTxHash(input.ClientRequest.SignedTransaction().Transaction())
	receiptChannel := make(chan *services.AddNewTransactionOutput)

	s.mutex.Lock()
	s.txChan[txId.KeyForMap()] = receiptChannel
	defer func() {
		s.mutex.Lock()
		delete(s.txChan, txId.KeyForMap())
		s.mutex.Unlock()
		close(receiptChannel)
	}()
	s.mutex.Unlock()

	timer := time.NewTimer(s.config.SendTransactionTimeout())
	defer timer.Stop()

	var ta *services.AddNewTransactionOutput
	select {
	case <-timer.C:
		return nil, errors.Errorf("timed out waiting for transaction result")
	case ta = <-receiptChannel:
	}
	return prepareResponse(ta), nil
}

func prepareResponse(transactionOutput *services.AddNewTransactionOutput) *services.SendTransactionOutput {
	receipt := transactionOutput.TransactionReceipt

	mabs := make([]*protocol.MethodArgumentBuilder, 0, 1)
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

	//TODO this is terrible, delete and make it right (shaiy)
	response := &client.SendTransactionResponseBuilder{
		TransactionReceipt: &protocol.TransactionReceiptBuilder{
			Txhash:          receipt.Txhash(),
			ExecutionResult: receipt.ExecutionResult(),
			OutputArguments: mabs,
		},
		TransactionStatus: transactionOutput.TransactionStatus,
		BlockHeight:       transactionOutput.BlockHeight,
		BlockTimestamp:    transactionOutput.BlockTimestamp,
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

func (s *service) HandleTransactionResults(input *handlers.HandleTransactionResultsInput) (*handlers.HandleTransactionResultsOutput, error) {
	panic("Not implemented")
}

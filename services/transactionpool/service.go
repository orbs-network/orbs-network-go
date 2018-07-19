package transactionpool

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

type service struct {
	pendingTransactions chan *protocol.SignedTransaction
	gossip              gossiptopics.TransactionRelay
	reporting           instrumentation.Reporting
}

func NewTransactionPool(gossip gossiptopics.TransactionRelay, reporting instrumentation.Reporting) services.TransactionPool {
	s := &service{
		pendingTransactions: make(chan *protocol.SignedTransaction, 10),
		gossip:              gossip,
		reporting:           reporting,
	}
	gossip.RegisterTransactionRelayHandler(s)
	return s
}

func (s *service) AddNewTransaction(input *services.AddNewTransactionInput) (*services.AddNewTransactionOutput, error) {
	s.reporting.Info(fmt.Sprintf("Adding new transaction [%v] to the pool", input.SignedTransaction))
	s.gossip.BroadcastForwardedTransactions(&gossiptopics.ForwardedTransactionsInput{
		Message: &gossipmessages.ForwardedTransactionsMessage{
			SignedTransactions: []*protocol.SignedTransaction{input.SignedTransaction},
		},
	})

	//This is commented out because currently transport broadcast will also broadcast to myself. So HandleForwardedTransactions will be the on to add this transaction.
	// no longer commented out
	s.pendingTransactions <- input.SignedTransaction

	return &services.AddNewTransactionOutput{}, nil
}

func (s *service) GetTransactionsForOrdering(input *services.GetTransactionsForOrderingInput) (*services.GetTransactionsForOrderingOutput, error) {
	out := &services.GetTransactionsForOrderingOutput{}
	out.SignedTransactions = make([]*protocol.SignedTransaction, input.MaxNumberOfTransactions)
	for i := uint32(0); i < input.MaxNumberOfTransactions; i++ {
		out.SignedTransactions[i] = <-s.pendingTransactions
	}
	return out, nil
}

func (s *service) GetCommittedTransactionReceipt(input *services.GetCommittedTransactionReceiptInput) (*services.GetCommittedTransactionReceiptOutput, error) {
	panic("Not implemented")
}

func (s *service) ValidateTransactionsForOrdering(input *services.ValidateTransactionsForOrderingInput) (*services.ValidateTransactionsForOrderingOutput, error) {
	panic("Not implemented")
}

func (s *service) CommitTransactionReceipts(input *services.CommitTransactionReceiptsInput) (*services.CommitTransactionReceiptsOutput, error) {
	panic("Not implemented")
}

func (s *service) RegisterTransactionResultsHandler(handler handlers.TransactionResultsHandler) {
	panic("Not implemented")
}

func (s *service) HandleForwardedTransactions(input *gossiptopics.ForwardedTransactionsInput) (*gossiptopics.EmptyOutput, error) {
	for _, tx := range input.Message.SignedTransactions {
		s.pendingTransactions <- tx
	}
	return nil, nil
}

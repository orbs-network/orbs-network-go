package transactionpool

import (
	"fmt"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

type service struct {
	services.TransactionPool
	pendingTransactions chan *protocol.SignedTransaction
	gossip gossiptopics.TransactionRelay
}

func NewTransactionPool(gossip gossiptopics.TransactionRelay) services.TransactionPool {
	s := &service{
		pendingTransactions: make(chan *protocol.SignedTransaction, 10),
		gossip : gossip,
	}
	gossip.RegisterTransactionRelayHandler(s)
	return s
}

func (s *service) AddNewTransaction(input *services.AddNewTransactionInput) (*services.AddNewTransactionOutput, error) {
	fmt.Println("Adding new transaction to the pool", input.SignedTransaction)
	s.gossip.BroadcastForwardedTransactions(&gossiptopics.ForwardedTransactionsInput{Transactions:[]*protocol.SignedTransaction{input.SignedTransaction}})
	//This is commented out because currently transport broadcast will also broadcast to myself. So HandleForwardedTransactions will be the on to add this transaction.
	//s.pendingTransactions <- input.SignedTransaction
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

// Deprecated: TransactionListener is going away in favor of TransactionRelayGossipHandler
func (s *service) OnForwardTransaction(tx *protocol.SignedTransaction) {
	s.pendingTransactions <- tx
}

func (s *service) HandleForwardedTransactions(input *gossiptopics.ForwardedTransactionsInput) (*gossiptopics.TransactionRelayOutput, error) {
	txs := input.Transactions
	for _, tx := range txs {
		s.pendingTransactions <- tx
	}
	return nil, nil
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
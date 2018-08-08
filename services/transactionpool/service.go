package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"sync"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
)

type txPool struct {
	transactions map[string]bool
	lock         *sync.Mutex
}

func (p txPool) add(transaction *protocol.SignedTransaction) {
	key := hash.CalcSha256(transaction.Raw()).KeyForMap()
	p.lock.Lock()
	defer p.lock.Unlock()
	p.transactions[key] = true
}

func (p txPool) has(transaction *protocol.SignedTransaction) bool {
	key := hash.CalcSha256(transaction.Raw()).KeyForMap()
	ok, _ := p.transactions[key]
	return ok
}

type service struct {
	pendingTransactions chan *protocol.SignedTransaction
	gossip              gossiptopics.TransactionRelay
	reporting           instrumentation.BasicLogger

	pendingPool txPool
}

func NewTransactionPool(gossip gossiptopics.TransactionRelay, reporting instrumentation.BasicLogger) services.TransactionPool {
	s := &service{
		pendingTransactions: make(chan *protocol.SignedTransaction, 10),
		gossip:              gossip,
		reporting:           reporting.For(instrumentation.Service("transaction-pool")),

		pendingPool: txPool{
			transactions: make(map[string]bool),
			lock:         &sync.Mutex{},
		},
	}
	gossip.RegisterTransactionRelayHandler(s)
	return s
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

func (s *service) isTransactionInPendingPool(transaction *protocol.SignedTransaction) bool {
	return s.pendingPool.has(transaction)
}

func (s *service) isTransactionInCommittedPool(transaction *protocol.SignedTransaction) bool {
	return false //TODO really check
}

package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"sync"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
)

type pendingTxPool struct {
	transactions map[string]bool
	lock         *sync.Mutex
}

func (p pendingTxPool) add(transaction *protocol.SignedTransaction) {
	key := digest.CalcTxHash(transaction.Transaction()).KeyForMap()
	p.lock.Lock()
	defer p.lock.Unlock()
	p.transactions[key] = true
}

func (p pendingTxPool) has(transaction *protocol.SignedTransaction) bool {
	key := digest.CalcTxHash(transaction.Transaction()).KeyForMap()
	ok, _ := p.transactions[key]
	return ok
}

func (p pendingTxPool) remove(txhash primitives.Sha256) {
	delete(p.transactions, txhash.KeyForMap())
}

type committedTxPool struct {
	transactions map[string]*committedTransaction
	lock         *sync.Mutex
}

func (p committedTxPool) add(receipt *protocol.TransactionReceipt) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.transactions[receipt.Txhash().KeyForMap()] = &committedTransaction{
		receipt: receipt,
	}
}

func (p committedTxPool) get(transaction *protocol.SignedTransaction) *committedTransaction {
	key := digest.CalcTxHash(transaction.Transaction()).KeyForMap()

	tx := p.transactions[key]

	return tx
}

type committedTransaction struct {
	receipt *protocol.TransactionReceipt
}

type service struct {
	pendingTransactions chan *protocol.SignedTransaction
	gossip              gossiptopics.TransactionRelay
	virtualMachine services.VirtualMachine
	reporting           instrumentation.BasicLogger

	pendingPool    pendingTxPool
	committedPool  committedTxPool
}

func NewTransactionPool(gossip gossiptopics.TransactionRelay, virtualMachine services.VirtualMachine, reporting instrumentation.BasicLogger) services.TransactionPool {
	s := &service{
		pendingTransactions: make(chan *protocol.SignedTransaction, 10),
		gossip:              gossip,
		virtualMachine:      virtualMachine,
		reporting:           reporting.For(instrumentation.Service("transaction-pool")),

		pendingPool: pendingTxPool{
			transactions: make(map[string]bool),
			lock:         &sync.Mutex{},
		},

		committedPool: committedTxPool{
			transactions: make(map[string]*committedTransaction),
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
	for _, receipt := range input.TransactionReceipts {
		s.committedPool.add(receipt)
		s.pendingPool.remove(receipt.Txhash())
	}

	return &services.CommitTransactionReceiptsOutput{}, nil
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

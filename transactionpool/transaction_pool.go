package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/gossip"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type TransactionPool interface {
	gossip.TransactionListener

	Add(tx *protocol.SignedTransaction)
	Next() *protocol.SignedTransaction
}

type inMemoryTransactionPool struct {
	pendingTransactions chan *protocol.SignedTransaction
}

func NewTransactionPool(gossip gossip.Gossip) TransactionPool {
	pool := &inMemoryTransactionPool{make(chan *protocol.SignedTransaction, 10)}
	gossip.RegisterTransactionListener(pool)
	return pool
}

func (p *inMemoryTransactionPool) Add(tx *protocol.SignedTransaction) {
	p.pendingTransactions <- tx
}

func (p *inMemoryTransactionPool) Next() *protocol.SignedTransaction {
	return <- p.pendingTransactions
}

func (p *inMemoryTransactionPool) OnForwardTransaction(tx *protocol.SignedTransaction) {
	p.pendingTransactions <- tx
}

package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/gossip"
	"github.com/orbs-network/orbs-network-go/types"
)

type TransactionPool interface {
	gossip.TransactionListener

	Add(tx *types.Transaction)
	Next() *types.Transaction
}

type inMemoryTransactionPool struct {
	pendingTransactions chan *types.Transaction
}

func NewTransactionPool(gossip gossip.Gossip) TransactionPool {
	pool := &inMemoryTransactionPool{make(chan *types.Transaction, 10)}
	gossip.RegisterTransactionListener(pool)
	return pool
}

func (p *inMemoryTransactionPool) Add(tx *types.Transaction) {
	p.pendingTransactions <- tx
}

func (p *inMemoryTransactionPool) Next() *types.Transaction {
	return <- p.pendingTransactions
}

func (p *inMemoryTransactionPool) OnForwardTransaction(tx *types.Transaction) {
	p.pendingTransactions <- tx
}

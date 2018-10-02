package transactionpool

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"sync"
	"time"
)

type committedTxPool struct {
	transactions map[string]*committedTransaction
	lock         *sync.RWMutex
}

func NewCommittedPool() *committedTxPool {
	return &committedTxPool{
		transactions: make(map[string]*committedTransaction),
		lock:         &sync.RWMutex{},
	}
}

type committedTransaction struct {
	receipt   *protocol.TransactionReceipt
	timestamp primitives.TimestampNano
}

func (p *committedTxPool) add(receipt *protocol.TransactionReceipt, ts primitives.TimestampNano) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.transactions[receipt.Txhash().KeyForMap()] = &committedTransaction{
		receipt:   receipt,
		timestamp: ts,
	}
}

func (p *committedTxPool) get(txHash primitives.Sha256) *committedTransaction {
	key := txHash.KeyForMap()

	p.lock.RLock()
	defer p.lock.RUnlock()

	tx := p.transactions[key]

	return tx
}

func (p *committedTxPool) has(txHash primitives.Sha256) bool {
	p.lock.RLock()
	defer p.lock.RUnlock()

	_, ok := p.transactions[txHash.KeyForMap()]
	return ok
}

func (p *committedTxPool) clearTransactionsOlderThan(time time.Time) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	for _, tx := range p.transactions {
		if int64(tx.timestamp) < time.UnixNano() {
			delete(p.transactions, tx.receipt.Txhash().KeyForMap())
		}
	}
}

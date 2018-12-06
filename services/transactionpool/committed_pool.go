package transactionpool

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"sync"
	"time"
)

type committedTxPool struct {
	transactions map[string]*committedTransaction
	lock         *sync.RWMutex

	metrics *committedPoolMetrics
}

type committedPoolMetrics struct {
	transactionCount *metric.Gauge
	poolSizeInBytes  *metric.Gauge
}

func newCommittedPoolMetrics(factory metric.Factory) *committedPoolMetrics {
	return &committedPoolMetrics{
		transactionCount: factory.NewGauge("TransactionPool.CommittedPool.TransactionCount"),
		poolSizeInBytes:  factory.NewGauge("TransactionPool.CommittedPool.PoolSizeInBytes"),
	}
}

func NewCommittedPool(metricFactory metric.Factory) *committedTxPool {
	return &committedTxPool{
		transactions: make(map[string]*committedTransaction),
		lock:         &sync.RWMutex{},
		metrics:      newCommittedPoolMetrics(metricFactory),
	}
}

type committedTransaction struct {
	receipt        *protocol.TransactionReceipt
	timestampAdded primitives.TimestampNano
	blockHeight    primitives.BlockHeight
	blockTimestamp primitives.TimestampNano
}

func (p *committedTxPool) add(receipt *protocol.TransactionReceipt, tsAdded primitives.TimestampNano, blockHeight primitives.BlockHeight, blockTs primitives.TimestampNano) {
	p.lock.Lock()
	defer p.lock.Unlock()

	transaction := &committedTransaction{
		receipt:        receipt,
		timestampAdded: tsAdded,
		blockHeight:    blockHeight,
		blockTimestamp: blockTs,
	}
	size := sizeOfCommittedTransaction(transaction)

	p.transactions[receipt.Txhash().KeyForMap()] = transaction

	p.metrics.transactionCount.Inc()
	p.metrics.poolSizeInBytes.AddUint32(size)
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

func (p *committedTxPool) clearTransactionsOlderThan(ctx context.Context, time time.Time) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	for _, tx := range p.transactions {
		if int64(tx.timestampAdded) < time.UnixNano() {
			delete(p.transactions, tx.receipt.Txhash().KeyForMap())

			p.metrics.transactionCount.Dec()
			p.metrics.poolSizeInBytes.SubUint32(sizeOfCommittedTransaction(tx))
		}
	}
}

// Excluding timestamps
func sizeOfCommittedTransaction(transaction *committedTransaction) uint32 {
	return uint32(len(transaction.receipt.Raw()))
}

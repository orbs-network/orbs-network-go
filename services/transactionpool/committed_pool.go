package transactionpool

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"sync"
)

type committedTxPool struct {
	sync.RWMutex
	transactions map[string]*committedTransaction

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
		metrics:      newCommittedPoolMetrics(metricFactory),
	}
}

type committedTransaction struct {
	receipt        *protocol.TransactionReceipt
	submitted      primitives.TimestampNano
	blockHeight    primitives.BlockHeight
	blockTimestamp primitives.TimestampNano
}

func (p *committedTxPool) add(receipt *protocol.TransactionReceipt, submitted primitives.TimestampNano, blockHeight primitives.BlockHeight, blockTs primitives.TimestampNano) {
	p.Lock()
	defer p.Unlock()

	transaction := &committedTransaction{
		receipt:        receipt,
		submitted:      submitted,
		blockHeight:    blockHeight,
		blockTimestamp: blockTs,
	}
	size := sizeOfCommittedTransaction(transaction)

	p.transactions[receipt.Txhash().KeyForMap()] = transaction

	p.metrics.transactionCount.Inc()
	p.metrics.poolSizeInBytes.AddUint32(size)
}

func (p *committedTxPool) get(txHash primitives.Sha256) *committedTransaction {
	p.RLock()
	defer p.RUnlock()

	return p.transactions[txHash.KeyForMap()]
}

func (p *committedTxPool) has(txHash primitives.Sha256) bool {
	p.RLock()
	defer p.RUnlock()

	_, ok := p.transactions[txHash.KeyForMap()]
	return ok
}

func (p *committedTxPool) clearTransactionsOlderThan(ctx context.Context, timestamp primitives.TimestampNano) {
	p.Lock()
	defer p.Unlock()

	for _, tx := range p.transactions {
		if tx.submitted < timestamp {
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

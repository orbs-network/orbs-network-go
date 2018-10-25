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
	transactionCountGauge *metric.Gauge
	poolSizeInBytesGauge  *metric.Gauge // TODO use this metric
}

func newCommittedPoolMetrics(factory metric.Factory) *committedPoolMetrics {
	return &committedPoolMetrics{
		transactionCountGauge: factory.NewGauge("TransactionPool.CommittedPool.TransactionCount"),
		poolSizeInBytesGauge:  factory.NewGauge("TransactionPool.CommittedPool.PoolSizeInBytes"),
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

	p.metrics.transactionCountGauge.Inc()
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
		if int64(tx.timestamp) < time.UnixNano() {
			delete(p.transactions, tx.receipt.Txhash().KeyForMap())
		}
	}
}

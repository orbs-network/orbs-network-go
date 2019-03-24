// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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
	sync.RWMutex
	transactions map[string]*committedTransaction

	transactionPoolFutureTimestampGraceTimeout func() time.Duration

	metrics *committedPoolMetrics
}

type committedPoolMetrics struct {
	transactionCount *metric.Gauge
	poolSizeInBytes  *metric.Gauge
}

func newCommittedPoolMetrics(factory metric.Factory) *committedPoolMetrics {
	return &committedPoolMetrics{
		transactionCount: factory.NewGauge("TransactionPool.CommittedPool.Transactions.Count"),
		poolSizeInBytes:  factory.NewGauge("TransactionPool.CommittedPool.PoolSize.Bytes"),
	}
}

func NewCommittedPool(transactionPoolFutureTimestampGraceTimeout func() time.Duration, metricFactory metric.Factory) *committedTxPool {
	return &committedTxPool{
		transactionPoolFutureTimestampGraceTimeout: transactionPoolFutureTimestampGraceTimeout,
		transactions: make(map[string]*committedTransaction),
		metrics:      newCommittedPoolMetrics(metricFactory),
	}
}

type committedTransaction struct {
	receipt        *protocol.TransactionReceipt
	blockHeight    primitives.BlockHeight
	blockTimestamp primitives.TimestampNano
}

func (p *committedTxPool) add(receipt *protocol.TransactionReceipt, blockHeight primitives.BlockHeight, blockTs primitives.TimestampNano) {
	p.Lock()
	defer p.Unlock()

	transaction := &committedTransaction{
		receipt:        receipt,
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

	futureTimestampGrace := primitives.TimestampNano(p.transactionPoolFutureTimestampGraceTimeout().Nanoseconds())

	for _, tx := range p.transactions {
		if tx.blockTimestamp+futureTimestampGrace < timestamp {
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

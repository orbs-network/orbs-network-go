// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package transactionpool

import (
	"container/list"
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"sync"
	"time"
)

type transactionRemovedListener func(ctx context.Context, txHash primitives.Sha256, reason protocol.TransactionStatus)

func NewPendingPool(pendingPoolSizeInBytes func() uint32, metricFactory metric.Factory, onNewTransaction func()) *pendingTxPool {
	return &pendingTxPool{
		pendingPoolSizeInBytes: pendingPoolSizeInBytes,
		transactionsByHash:     make(map[string]*pendingTransaction),
		transactionList:        list.New(),
		lock:                   &sync.RWMutex{},
		onNewTransaction:       onNewTransaction,

		metrics: newPendingPoolMetrics(metricFactory),
	}
}

type pendingTransaction struct {
	gatewayNodeAddress primitives.NodeAddress
	transaction        *protocol.SignedTransaction
	listElement        *list.Element
	timeAdded          time.Time
}

type pendingPoolMetrics struct {
	transactionCountGauge    *metric.Gauge
	poolSizeInBytesGauge     *metric.Gauge
	transactionRatePerSecond *metric.Rate
	transactionSpentInQueue  *metric.Histogram
	transactionServiceTime   *metric.Histogram
}

func newPendingPoolMetrics(factory metric.Factory) *pendingPoolMetrics {
	return &pendingPoolMetrics{
		transactionServiceTime:   factory.NewLatency("TransactionPool.ServiceTime.Millis", 30*time.Minute),
		transactionCountGauge:    factory.NewGauge("TransactionPool.PendingPool.Transactions.Count"),
		poolSizeInBytesGauge:     factory.NewGauge("TransactionPool.PendingPool.PoolSize.Bytes"),
		transactionRatePerSecond: factory.NewRate("TransactionPool.TransactionsEnteringPool.PerSecond"),
		transactionSpentInQueue:  factory.NewLatency("TransactionPool.PendingPool.TimeSpentInQueue.Millis", 30*time.Minute),
	}
}

type pendingTxPool struct {
	currentSizeInBytes uint32
	transactionsByHash map[string]*pendingTransaction
	transactionList    *list.List
	onNewTransaction   func()
	lock               *sync.RWMutex

	pendingPoolSizeInBytes func() uint32
	onTransactionRemoved   transactionRemovedListener

	metrics *pendingPoolMetrics
}

func (p *pendingTxPool) add(transaction *protocol.SignedTransaction, gatewayNodeAddress primitives.NodeAddress) (primitives.Sha256, *ErrTransactionRejected) {
	size := sizeOfSignedTransaction(transaction)

	if p.currentSizeInBytes+size > p.pendingPoolSizeInBytes() {
		return nil, &ErrTransactionRejected{TransactionStatus: protocol.TRANSACTION_STATUS_REJECTED_CONGESTION}
	}

	key := digest.CalcTxHash(transaction.Transaction())

	p.lock.Lock()
	defer p.lock.Unlock()
	if _, exists := p.transactionsByHash[key.KeyForMap()]; exists {
		return nil, &ErrTransactionRejected{TransactionStatus: protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING}
	}

	p.currentSizeInBytes += size
	p.transactionsByHash[key.KeyForMap()] = &pendingTransaction{
		transaction:        transaction,
		gatewayNodeAddress: gatewayNodeAddress,
		listElement:        p.transactionList.PushFront(transaction),
		timeAdded:          time.Now(),
	}

	p.metrics.transactionCountGauge.Inc()
	p.metrics.poolSizeInBytesGauge.AddUint32(size)
	p.metrics.transactionRatePerSecond.Measure(1)

	p.onNewTransaction()

	return key, nil
}

func (p *pendingTxPool) has(transaction *protocol.SignedTransaction) bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	key := digest.CalcTxHash(transaction.Transaction()).KeyForMap()
	_, ok := p.transactionsByHash[key]
	return ok
}

func (p *pendingTxPool) remove(ctx context.Context, txHash primitives.Sha256, removalReason protocol.TransactionStatus) *primitives.NodeAddress {
	p.lock.Lock()
	defer p.lock.Unlock()

	pendingTx, ok := p.transactionsByHash[txHash.KeyForMap()]
	if ok {
		delete(p.transactionsByHash, txHash.KeyForMap())
		p.currentSizeInBytes -= sizeOfSignedTransaction(pendingTx.transaction)
		p.transactionList.Remove(pendingTx.listElement)

		if p.onTransactionRemoved != nil {
			p.onTransactionRemoved(ctx, txHash, removalReason)
		}

		p.metrics.transactionCountGauge.Dec()
		p.metrics.poolSizeInBytesGauge.SubUint32(sizeOfSignedTransaction(pendingTx.transaction))
		p.metrics.transactionServiceTime.RecordSince(pendingTx.timeAdded)

		return &pendingTx.gatewayNodeAddress
	}

	return nil
}

func (p *pendingTxPool) getBatch(maxNumOfTransactions uint32, sizeLimitInBytes uint32) (txs Transactions) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	var sizeInBytes uint32

	e := p.transactionList.Back()
	for {
		if e == nil {
			break
		}

		if uint32(len(txs)) >= maxNumOfTransactions {
			break
		}

		tx := e.Value.(*protocol.SignedTransaction)
		//
		txSize := sizeOfSignedTransaction(tx)
		if sizeLimitInBytes > 0 && sizeInBytes+txSize > sizeLimitInBytes {
			break
		}

		sizeInBytes += txSize
		txs = append(txs, tx)

		e = e.Prev()

		p.transactionPickedFromQueueUnderMutex(tx)
	}

	return
}

func (p *pendingTxPool) get(txHash primitives.Sha256) *protocol.SignedTransaction {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if ptx, ok := p.transactionsByHash[txHash.KeyForMap()]; ok {
		return ptx.transaction
	}

	return nil
}

func (p *pendingTxPool) clearTransactionsOlderThan(ctx context.Context, timestamp primitives.TimestampNano) {
	p.lock.RLock()
	e := p.transactionList.Back()
	p.lock.RUnlock()

	for {
		if e == nil {
			break
		}

		tx := e.Value.(*protocol.SignedTransaction)

		e = e.Prev()

		if tx.Transaction().Timestamp() < timestamp {
			p.remove(ctx, digest.CalcTxHash(tx.Transaction()), protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_WINDOW_EXCEEDED)
		}
	}
}

func (p *pendingTxPool) transactionPickedFromQueueUnderMutex(tx *protocol.SignedTransaction) {
	txHash := digest.CalcTxHash(tx.Transaction())
	ptx, found := p.transactionsByHash[txHash.KeyForMap()]
	if found {
		p.metrics.transactionSpentInQueue.RecordSince(ptx.timeAdded)
	}
}

func sizeOfSignedTransaction(transaction *protocol.SignedTransaction) uint32 {
	return uint32(len(transaction.Raw()))
}

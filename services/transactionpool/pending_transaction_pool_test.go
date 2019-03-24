// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package transactionpool

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

var nodeAddress = keys.EcdsaSecp256K1KeyPairForTests(8).NodeAddress()

func TestPendingTransactionPoolTracksSizesOfTransactionsAddedAndRemoved(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		p := makePendingPool()
		require.Zero(t, p.currentSizeInBytes, "New pending pool created with non-zero size")

		tx1 := builders.TransferTransaction().Build()
		k1, _ := p.add(tx1, nodeAddress)
		require.Equal(t, uint32(len(tx1.Raw())), p.currentSizeInBytes, "pending pool size did not reflect tx1 size")

		tx2 := builders.TransferTransaction().WithContract("a contract with a long name so that tx has a different size").Build()
		k2, _ := p.add(tx2, nodeAddress)
		require.Equal(t, uint32(len(tx1.Raw())+len(tx2.Raw())), p.currentSizeInBytes, "pending pool size did not reflect combined sizes of tx1 + tx2")

		p.remove(ctx, k1, 0)
		require.Equal(t, uint32(len(tx2.Raw())), p.currentSizeInBytes, "pending pool size did not reflect removal of tx1")

		p.remove(ctx, k2, 0)
		require.Zero(t, p.currentSizeInBytes, "pending pool size did not reflect removal of tx2")
	})
}

func TestPendingTransactionPoolAddRemoveKeepsBothDataStructuresInSync(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		p := makePendingPool()
		tx1 := builders.TransferTransaction().Build()

		k, _ := p.add(tx1, nodeAddress)
		require.True(t, p.has(tx1), "has() returned false for an added item")
		require.Len(t, p.getBatch(1, 0), 1, "getBatch() did not return an added item")

		p.remove(ctx, k, 0)
		require.False(t, p.has(tx1), "has() returned true for removed item")
		require.Empty(t, p.getBatch(1, 0), "getBatch() returned a removed item")

		require.NotPanics(t, func() {
			p.remove(ctx, k, 0)
		}, "removing a key that does not exist resulted in a panic")
	})
}

func TestPendingTransactionPoolGetBatchReturnsLessThanMaximumIfPoolHasLessTransaction(t *testing.T) {
	p := makePendingPool()

	add(p, builders.TransferTransaction().Build(), builders.TransferTransaction().Build())

	txSet := p.getBatch(3, 0)

	require.Len(t, txSet, 2, "expected 2 transactions but got %v transactions: %s", len(txSet), txSet)
}

func TestPendingTransactionPoolGetBatchDoesNotExceedSizeLimitInBytes(t *testing.T) {
	p := makePendingPool()

	tx1 := builders.TransferTransaction().Build()
	tx2 := builders.TransferTransaction().Build()
	add(p, tx1, tx2, builders.TransferTransaction().Build())

	slightlyMoreThanTwoTransactionsInBytes := uint32(len(tx1.Raw()) + len(tx2.Raw()) + 1)
	txSet := p.getBatch(3, slightlyMoreThanTwoTransactionsInBytes)

	require.Len(t, txSet, 2, "expected 2 transactions but got %v transactions: %s", len(txSet), txSet)
}

func TestPendingTransactionPoolGetBatchDoesNotExceedLengthLimit(t *testing.T) {
	p := makePendingPool()

	tx1 := builders.TransferTransaction().Build()
	tx2 := builders.TransferTransaction().Build()
	tx3 := builders.TransferTransaction().Build()
	add(p, tx1, tx2, tx3)

	txSet := p.getBatch(2, 0)

	require.Len(t, txSet, 2, "expected 2 transactions but got %v transactions: %s", len(txSet), txSet)
}

func TestPendingTransactionPoolGetBatchRetainsInsertionOrder(t *testing.T) {
	p := makePendingPool()

	// create 50 transactions so as to minimize the chance of randomly returning transactions in the expected order
	transactions := make(Transactions, 50)
	for i := 0; i < len(transactions); i++ {
		transactions[i] = builders.TransferTransaction().Build()
		add(p, transactions[i])
	}

	txSet := p.getBatch(uint32(len(transactions)), 0)

	require.Equal(t, transactions, txSet, "got transactions in wrong order")
}

func TestPendingTransactionPoolClearsExpiredTransactions(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		p := makePendingPool()

		tx1 := builders.TransferTransaction().WithTimestamp(time.Now().Add(-5 * time.Minute)).Build()
		tx2 := builders.TransferTransaction().WithTimestamp(time.Now().Add(-29 * time.Minute)).Build()
		tx3 := builders.TransferTransaction().WithTimestamp(time.Now().Add(-31 * time.Minute)).Build()
		add(p, tx1, tx2, tx3)

		p.clearTransactionsOlderThan(ctx, primitives.TimestampNano(time.Now().Add(-30*time.Minute).UnixNano()))

		require.True(t, p.has(tx1), "cleared non-expired transaction")
		require.True(t, p.has(tx2), "cleared non-expired transaction")
		require.False(t, p.has(tx3), "did not clear expired transaction")
	})
}

func TestPendingTransactionPoolDoesNotAddTheSameTransactionTwiceRegardlessOfPublicKey(t *testing.T) {
	p := makePendingPool()

	tx := builders.Transaction().Build()

	_, err := p.add(tx, nodeAddress)
	require.Nil(t, err, "got an unexpected error adding the first transaction")

	_, err = p.add(tx, nodeAddress)
	require.Equal(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING, err.TransactionStatus, "did not get expected status code")

	someOtherAddress := keys.EcdsaSecp256K1KeyPairForTests(3).NodeAddress()
	_, err = p.add(tx, someOtherAddress)
	require.Equal(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING, err.TransactionStatus, "did not get expected status code")

}

func TestPendingTransactionPoolCallsRemovalListenerWhenRemovingTransaction(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		var removedTxHash primitives.Sha256
		var removalReason protocol.TransactionStatus

		p := makePendingPool()
		p.onTransactionRemoved = func(ctx context.Context, txHash primitives.Sha256, reason protocol.TransactionStatus) {
			removedTxHash = txHash
			removalReason = reason
		}

		tx := builders.Transaction().Build()
		p.add(tx, nodeAddress)
		txHash := digest.CalcTxHash(tx.Transaction())
		p.remove(ctx, txHash, protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_WINDOW_EXCEEDED)

		require.Equal(t, txHash, removedTxHash, "removed txHash didn't equal expected txHash")
		require.Equal(t, protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_WINDOW_EXCEEDED, removalReason, "removal reason didn't equal expected reason")
	})
}

func TestPendingPoolNotifiesOnNewTransactions(t *testing.T) {
	var called bool
	p := NewPendingPool(func() uint32 { return 100000 }, metric.NewRegistry(), func() {
		called = true
	})

	p.add(builders.Transaction().Build(), nodeAddress)

	require.True(t, called, "pending transaction pool did not notify onNewTransaction")
}

func add(p *pendingTxPool, txs ...*protocol.SignedTransaction) {
	for _, tx := range txs {
		p.add(tx, nodeAddress)
	}
}

func makePendingPool() *pendingTxPool {
	metricFactory := metric.NewRegistry()
	return NewPendingPool(func() uint32 { return 100000 }, metricFactory, func() {})
}

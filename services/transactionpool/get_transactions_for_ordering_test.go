// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package transactionpool

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTransactionBatchFetchesUpToMaxNumOfTransactions(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		tx1 := builders.TransferTransaction().Build()
		tx2 := builders.TransferTransaction().Build()
		tx3 := builders.TransferTransaction().Build()

		b := &transactionBatch{
			logger:               log.DefaultTestingLogger(t),
			maxNumOfTransactions: 2,
		}

		f := &fakeFetcher{
			transactions: Transactions{tx1, tx2, tx3},
		}

		b.fetchUsing(f)

		require.Len(t, b.incomingTransactions, 2, "did not fetch exactly 2 transactions")
	})
}

func TestTransactionBatchFetchesUpToSizeLimit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		tx1 := builders.TransferTransaction().Build()
		tx2 := builders.TransferTransaction().Build()
		tx3 := builders.TransferTransaction().Build()

		b := &transactionBatch{
			logger:               log.DefaultTestingLogger(t),
			sizeLimit:            sizeOf(tx1, tx2) + 1,
			maxNumOfTransactions: 100,
		}

		f := &fakeFetcher{
			transactions: Transactions{tx1, tx2, tx3},
		}

		b.fetchUsing(f)

		require.Len(t, b.incomingTransactions, 2, "did not fetch exactly 2 transactions")
	})
}

func TestTransactionBatchRejectsTransactionsFailingStaticValidation(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		tx1 := builders.TransferTransaction().Build()
		tx2 := builders.TransferTransaction().Build()

		b := newTransactionBatch(log.DefaultTestingLogger(t), Transactions{tx1, tx2})
		b.filterInvalidTransactions(ctx, &fakeValidator{invalid: Transactions{tx2}}, &fakeCommittedChecker{}, 0)

		require.Empty(t, b.incomingTransactions, "did not empty incoming transaction list")

		require.Len(t, b.transactionsForPreOrder, 1)
		require.Equal(t, tx1, b.transactionsForPreOrder[0], "valid transaction was rejected")

		require.Equal(t, protocol.TRANSACTION_STATUS_RESERVED, b.transactionsToReject[0].status, "invalid transaction was not rejected")
		require.Equal(t, digest.CalcTxHash(tx2.Transaction()), b.transactionsToReject[0].hash, "invalid transaction was not rejected")
	})
}

func TestTransactionBatchRejectsCommittedTransaction(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		tx1 := builders.TransferTransaction().Build()
		tx2 := builders.TransferTransaction().Build()

		b := newTransactionBatch(log.DefaultTestingLogger(t), Transactions{tx1, tx2})
		b.filterInvalidTransactions(ctx, &fakeValidator{}, &fakeCommittedChecker{Transactions{tx2}}, 0)

		require.Empty(t, b.incomingTransactions, "did not empty incoming transaction list")

		require.Len(t, b.transactionsForPreOrder, 1)
		require.Equal(t, tx1, b.transactionsForPreOrder[0], "valid transaction was rejected")

		require.Equal(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, b.transactionsToReject[0].status, "invalid transaction was not rejected")
		require.Equal(t, digest.CalcTxHash(tx2.Transaction()), b.transactionsToReject[0].hash, "invalid transaction was not rejected")
	})
}

func TestTransactionBatchRejectsTransactionsFailingPreOrderValidation(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		tx1 := builders.TransferTransaction().Build()
		tx2 := builders.TransferTransaction().Build()
		tx3 := builders.TransferTransaction().Build()

		b := &transactionBatch{transactionsForPreOrder: Transactions{tx1, tx2, tx3}, logger: log.DefaultTestingLogger(t)}
		err := b.runPreOrderValidations(ctx, &fakeValidator{statuses: []protocol.TransactionStatus{protocol.TRANSACTION_STATUS_PRE_ORDER_VALID, protocol.TRANSACTION_STATUS_REJECTED_GLOBAL_PRE_ORDER, protocol.TRANSACTION_STATUS_REJECTED_SIGNATURE_MISMATCH}}, 0, 0)

		require.NoError(t, err, "this should really never happen")
		require.Empty(t, b.transactionsForPreOrder, "did not empty transaction for preorder list")

		require.Len(t, b.validTransactions, 1)
		require.Equal(t, tx1, b.validTransactions[0], "valid transaction was rejected")

		require.Equal(t, protocol.TRANSACTION_STATUS_REJECTED_GLOBAL_PRE_ORDER, b.transactionsToReject[0].status, "invalid transaction was not rejected")
		require.Equal(t, digest.CalcTxHash(tx2.Transaction()), b.transactionsToReject[0].hash, "invalid transaction was not rejected")

		require.Equal(t, protocol.TRANSACTION_STATUS_REJECTED_SIGNATURE_MISMATCH, b.transactionsToReject[1].status, "invalid transaction was not rejected")
		require.Equal(t, digest.CalcTxHash(tx3.Transaction()), b.transactionsToReject[1].hash, "invalid transaction was not rejected")
	})
}

func TestTransactionBatchPanicsIfPreOrderResultsHasDifferentLengthThanSent(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		tx1 := builders.TransferTransaction().Build()
		tx2 := builders.TransferTransaction().Build()

		b := &transactionBatch{transactionsForPreOrder: Transactions{tx1, tx2}, logger: log.DefaultTestingLogger(t)}
		require.Panics(t, func() {
			b.runPreOrderValidations(ctx, &fakeValidator{}, 0, 0)
		}, "pre order validation returning statuses with length that differs from number of txs sent did not panic")
	})
}

func TestTransactionBatchNotifiesOnRejectedTransactions(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h1 := digest.CalcTxHash(builders.TransferTransaction().Build().Transaction())
		h2 := digest.CalcTxHash(builders.TransferTransaction().Build().Transaction())

		b := &transactionBatch{transactionsToReject: []*rejectedTransaction{
			{hash: h1, status: protocol.TRANSACTION_STATUS_REJECTED_SMART_CONTRACT_PRE_ORDER},
			{hash: h2, status: protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_WINDOW_EXCEEDED},
		}}

		r := &fakeRemover{removed: make(map[string]protocol.TransactionStatus)}

		b.notifyRejections(ctx, r)

		require.Empty(t, b.transactionsToReject, "did not empty transactions to reject")
		require.Equal(t, protocol.TRANSACTION_STATUS_REJECTED_SMART_CONTRACT_PRE_ORDER, r.removed[h1.KeyForMap()])
		require.Equal(t, protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_WINDOW_EXCEEDED, r.removed[h2.KeyForMap()])
	})
}

type fakeValidator struct {
	statuses []protocol.TransactionStatus
	invalid  Transactions
}

func (v *fakeValidator) preOrderCheck(ctx context.Context, txs Transactions, currentBlockHeight primitives.BlockHeight, currentBlockTimestamp primitives.TimestampNano) ([]protocol.TransactionStatus, error) {
	return v.statuses, nil
}

type fakeCommittedChecker struct {
	committed Transactions
}

func (c *fakeCommittedChecker) has(txHash primitives.Sha256) bool {
	for _, tx := range c.committed {
		if digest.CalcTxHash(tx.Transaction()).Equal(txHash) {
			return true
		}
	}

	return false
}

func (v *fakeValidator) ValidateTransactionForOrdering(txToValidate *protocol.SignedTransaction, proposedBlockTimestamp primitives.TimestampNano) *ErrTransactionRejected {
	for _, tx := range v.invalid {
		if tx == txToValidate {
			return &ErrTransactionRejected{TransactionStatus: protocol.TRANSACTION_STATUS_RESERVED}
		}
	}

	return nil
}

type fakeRemover struct {
	removed map[string]protocol.TransactionStatus
}

func (r *fakeRemover) remove(ctx context.Context, txHash primitives.Sha256, removalReason protocol.TransactionStatus) *primitives.NodeAddress {
	r.removed[txHash.KeyForMap()] = removalReason
	return nil
}

type fakeFetcher struct {
	transactions Transactions
}

func (f *fakeFetcher) getBatch(maxNumOfTransactions uint32, sizeLimitInBytes uint32) (txs Transactions) {
	var totalSize uint32
	for i, tx := range f.transactions {
		if sizeLimitInBytes > 0 && totalSize+sizeOf(tx) > sizeLimitInBytes {
			break
		}
		if uint32(i) == maxNumOfTransactions {
			break
		}
		txs = append(txs, tx)
		totalSize += sizeOfSignedTransaction(tx)
	}

	f.transactions = f.transactions[:len(txs)]
	return
}

func sizeOf(transactions ...*protocol.SignedTransaction) (size uint32) {
	for _, tx := range transactions {
		size += sizeOfSignedTransaction(tx)
	}
	return
}

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

func TestCommitTransactionReceiptsRequestsNextBlockOnMismatch(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t).start(ctx)

		h.assumeBlockStorageAtHeight(0) // so that we report transactions for block 1
		out, err := h.reportTransactionsAsCommitted(ctx)
		require.NoError(t, err, "CommitTransactionReceipts returned an error when expecting next desired block height")
		require.EqualValues(t, 2, out.NextDesiredBlockHeight, "expected next desired block height to be 2")

		h.assumeBlockStorageAtHeight(3) // so that we report transactions for block 4
		out, err = h.reportTransactionsAsCommitted(ctx)
		require.NoError(t, err, "CommitTransactionReceipts returned an error when expecting next desired block height")
		require.EqualValues(t, 2, out.NextDesiredBlockHeight, "expected next desired block height to be 2")

		h.ignoringTransactionResults()

		require.NoError(t, h.verifyMocks())
	})
}

func TestStress_AddingSameTransactionMultipleTimesWhileReportingAsCommitted(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		const CONCURRENCY_COUNT = 500
		duplicateStatuses := []protocol.TransactionStatus{
			protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED,
			protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING}
		startBarrier := sync.WaitGroup{}
		doneBarrier := sync.WaitGroup{}
		startBarrier.Add(CONCURRENCY_COUNT)
		doneBarrier.Add(CONCURRENCY_COUNT)

		h := newHarness(t).allowingErrorsMatching("error adding transaction to pending pool").start(ctx)
		h.ignoringForwardMessages()
		h.ignoringTransactionResults()
		h.ignoringBlockHeightChecks()

		outerCtx := trace.NewContext(ctx, "outer")
		tx := builders.TransferTransaction().Build()
		_, err := h.addNewTransaction(outerCtx, tx)
		require.NoError(t, err, "adding a transaction returned an unexpected error")

		for i := 0; i < CONCURRENCY_COUNT; i++ { // the idea here is to bombard addNewTransaction with a lot of concurrent calls for the same tx while simultaneously reporting tx as committed
			go func() {
				innerCtx := trace.NewContext(ctx, "inner")
				startBarrier.Done() // release one count, I'm ready for test
				startBarrier.Wait() // wait for the others

				receipt, _ := h.addNewTransaction(innerCtx, tx) // concurrently with h.reportTransactionsAsCommitted()

				assert.Contains(t, duplicateStatuses, receipt.TransactionStatus, "transaction must receive a duplicate status while in transit between pending and committed pools")
				doneBarrier.Done()
			}()
		}

		startBarrier.Wait() // wait until all goroutines are at the barrier so we can release all at the same time to tease out any flakiness

		_, err = h.reportTransactionsAsCommitted(outerCtx, tx) // concurrently with h.addNewTransaction()
		require.NoError(t, err, "committing a transaction returned an unexpected error")

		doneBarrier.Wait()
	})
}

func TestCommitTransactionReceiptForTxThatWasNeverInPendingPool_ShouldCommitItAnyway(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t).start(ctx)
		tx := builders.TransferTransaction().Build()

		h.reportTransactionsAsCommitted(ctx, tx)

		output, err := h.getTxReceipt(ctx, tx)
		require.NoError(t, err, "could not get output for tx committed without adding it to pending pool")
		require.NotNil(t, output)
		require.Equal(t, protocol.TRANSACTION_STATUS_COMMITTED.String(), output.TransactionStatus.String(), "transaction was not committed")
		require.NotNil(t, output.TransactionReceipt, "transaction was not committed")

		require.NoError(t, h.verifyMocks(), "Mocks were not executed as planned")
	})
}

func TestCommitTransactionReceiptsIgnoresExpiredBlocks(t *testing.T) {
	t.Skipf("TODO(v1): ignore blocks with an expired timestamp")
}

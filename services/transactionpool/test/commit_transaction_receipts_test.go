package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCommitTransactionReceiptsRequestsNextBlockOnMismatch(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(ctx)

		h.assumeBlockStorageAtHeight(3)
		out, err := h.reportTransactionsAsCommitted(ctx)
		require.NoError(t, err, "CommitTransactionReceipts returned an error when expecting next desired block height")
		require.EqualValues(t, 2, out.NextDesiredBlockHeight, "expected next desired block height to be 2")

		h.ignoringTransactionResults()

		h.assumeBlockStorageAtHeight(1)
		out, err = h.reportTransactionsAsCommitted(ctx)
		require.NoError(t, err, "CommitTransactionReceipts returned an error when expecting next desired block height")
		require.EqualValues(t, 2, out.NextDesiredBlockHeight, "expected next desired block height to be 2")

		h.verifyMocks()
	})
}

func TestCommitTransactionReceiptsNotifiesPublicAPIOnlyForOwnTransactions(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(ctx)
		myTx1 := builders.TransferTransaction().Build()
		myTx2 := builders.TransferTransaction().Build()
		otherTx := builders.TransferTransaction().Build()

		h.ignoringForwardMessages()

		h.addNewTransaction(ctx, myTx1)
		h.addNewTransaction(ctx, myTx2)
		h.handleForwardFrom(ctx, otherNodeKeyPair, otherTx)

		h.assumeBlockStorageAtHeight(1)
		h.expectTransactionResultsCallbackFor(myTx1, myTx2)
		h.reportTransactionsAsCommitted(ctx, myTx1, myTx2, otherTx)

		require.NoError(t, h.verifyMocks(), "Mocks were not executed as planned")
	})
}

func TestCommitTransactionReceiptsIgnoresExpiredBlocks(t *testing.T) {
	t.Skipf("TODO: ignore blocks with an expired timestamp")
}

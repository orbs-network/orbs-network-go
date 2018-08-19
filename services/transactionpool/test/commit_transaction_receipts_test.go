package test

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCommitTransactionReceiptsRequestsNextBlockOnMismatch(t *testing.T) {
	t.Parallel()
	h := newHarness()

	h.assumeBlockStorageAtHeight(3)
	out, err := h.reportTransactionsAsCommitted()
	require.NoError(t, err, "CommitTransactionReceipts returned an error when expecting next desired block height")
	require.EqualValues(t, 1, out.NextDesiredBlockHeight, "expected next desired block height to be 1")

	h.ignoringTransactionResults()

	h.assumeBlockStorageAtHeight(1)
	out, err = h.reportTransactionsAsCommitted()
	require.NoError(t, err, "CommitTransactionReceipts returned an error when expecting next desired block height")
	require.EqualValues(t, 2, out.NextDesiredBlockHeight, "expected next desired block height to be 2")

	h.verifyMocks()
}

func TestCommitTransactionReceiptsNotifiesPublicAPIOnlyForOwnTransactions(t *testing.T) {
	t.Parallel()
	h := newHarness()
	myTx1 := builders.TransferTransaction().Build()
	myTx2 := builders.TransferTransaction().Build()
	otherTx := builders.TransferTransaction().Build()

	h.ignoringForwardMessages()

	h.addNewTransaction(myTx1)
	h.addNewTransaction(myTx2)
	h.handleForwardFrom(otherNodeKeyPair, otherTx)

	h.assumeBlockStorageAtHeight(1)
	h.expectTransactionResultsCallbackFor(myTx1, myTx2)
	h.reportTransactionsAsCommitted(myTx1, myTx2, otherTx)

	require.NoError(t, h.verifyMocks(), "Mocks were not executed as planned")
}

func TestCommitTransactionReceiptsIgnoresExpiredBlocks(t *testing.T) {
	t.Skipf("TODO: ignore blocks with an expired timestamp")
}

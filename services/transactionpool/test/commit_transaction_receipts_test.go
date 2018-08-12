package test

import (
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCommitTransactionReceiptsRequestsNextBlockOnMismatch(t *testing.T) {
	h := newHarness()

	out, err := h.txpool.CommitTransactionReceipts(&services.CommitTransactionReceiptsInput{
		LastCommittedBlockHeight: 3,
	})

	require.NoError(t, err, "CommitTransactionReceipts returned an error when expecting next desired block height")
	require.EqualValues(t, 1, out.NextDesiredBlockHeight, "expected next desired block height to be 1")

	out, err = h.txpool.CommitTransactionReceipts(&services.CommitTransactionReceiptsInput{
		LastCommittedBlockHeight: 1,
	})

	require.NoError(t, err, "CommitTransactionReceipts returned an error when expecting next desired block height")
	require.EqualValues(t, 2, out.NextDesiredBlockHeight, "expected next desired block height to be 2")

}

//func TestCommitTransactionReceiptsNotifiesPublicAPIOnlyForOwnTransactions(t *testing.T) {
//	h := NewHarness()
//	myTx := builders.TransferTransaction().Build()
//	otherTx := builders.TransferTransaction().Build()
//
//	h.addNewTransaction(myTx)
//	h.handleForwardFrom(tx)
//
//}

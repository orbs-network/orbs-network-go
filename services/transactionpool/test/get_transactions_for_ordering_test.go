package test

import (
	"testing"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"time"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

//TODO does not include expired transactions
//TODO blocks and waits for grace (use blocktracker?)
//TODO fails for block too far away
//TODO num of transactions does not exceed limit
//TODO size of transaction set does not exceed limit
//TODO does not return already committed transactions

func TestGetTransactionsForOrderingReturnsAFIFOTransactionSet(t *testing.T) {
	h := newHarness()
	h.ignoringForwardMessages()

	now := time.Now()
	tx1 := builders.TransferTransaction().WithTimestamp(now.Add(-3 * time.Second)).Build()
	tx2 := builders.TransferTransaction().WithTimestamp(now.Add(-2 * time.Second)).Build()
	tx3 := builders.TransferTransaction().WithTimestamp(now.Add(-4 * time.Second)).Build()

	h.addTransactions(tx1, tx2, tx3)

	txSet, err := h.txpool.GetTransactionsForOrdering(&services.GetTransactionsForOrderingInput{
		MaxNumberOfTransactions: 3,
		MaxTransactionsSetSizeKb: 100,
		BlockHeight: 1,
	})

	require.NoError(t, err, "expected transaction set but got an error")
	require.Len(t, txSet.SignedTransactions, 3, "expected 3 transactions but got %v transactions: %s", len(txSet.SignedTransactions), txSet)
	require.Equal(t, []*protocol.SignedTransaction{tx3, tx1, tx2}, txSet.SignedTransactions, "got transaction set in wrong order")
}


package test

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

//TODO blocks and waits for grace (use blocktracker?)
//TODO fails for block too far away
//TODO does not return already committed transactions

func TestGetTransactionsForOrderingReturnsAFIFOTransactionSet(t *testing.T) {
	t.Parallel()
	h := newHarness()
	h.ignoringForwardMessages()

	now := time.Now()
	tx1 := builders.TransferTransaction().WithTimestamp(now.Add(-3 * time.Second)).Build()
	tx2 := builders.TransferTransaction().WithTimestamp(now.Add(-2 * time.Second)).Build()
	tx3 := builders.TransferTransaction().WithTimestamp(now.Add(-4 * time.Second)).Build()

	h.addTransactions(tx1, tx2, tx3)

	txSet, err := h.getTransactionsForOrdering(3)

	require.NoError(t, err, "expected transaction set but got an error")
	require.Len(t, txSet.SignedTransactions, 3, "expected 3 transactions but got %v transactions: %s", len(txSet.SignedTransactions), txSet)
	require.Equal(t, []*protocol.SignedTransaction{tx3, tx1, tx2}, txSet.SignedTransactions, "got transaction set in wrong order")
}

func TestGetTransactionsForOrderingDropsExpiredTransactions(t *testing.T) {
	t.Parallel()
	h := newHarness()

	validTx := builders.TransferTransaction().Build()
	expiredTx := builders.TransferTransaction().WithTimestamp(time.Now().Add(-1 * time.Duration(transactionExpirationWindowInSeconds + 60) * time.Second)).Build()

	// we use forward rather than add to simulate a scenario where a byzantine node submitted invalid transactions
	h.handleForwardFrom(otherNodeKeyPair.PublicKey(), validTx, expiredTx)

	txSet, err := h.getTransactionsForOrdering(2)

	require.NoError(t, err, "expected transaction set but got an error")
	require.Equal(t, []*protocol.SignedTransaction{validTx}, txSet.SignedTransactions, "got an expired transaction")
}


func TestGetTransactionsForOrderingDropTransactionsThatFailPreOrderValidation(t *testing.T) {
	t.Parallel()
	h := newHarness()
	h.ignoringForwardMessages()

	tx1 := builders.TransferTransaction().Build()
	tx2 := builders.TransferTransaction().Build()
	tx3 := builders.TransferTransaction().Build()
	tx4 := builders.TransferTransaction().Build()

	h.addTransactions(tx1, tx2, tx3, tx4)

	h.failPreOrderCheckFor(func(tx *protocol.SignedTransaction) bool {
		return tx == tx1 || tx == tx3
	});

	txSet, err := h.getTransactionsForOrdering(4)

	require.NoError(t, err, "expected transaction set but got an error")
	require.ElementsMatch(t, []*protocol.SignedTransaction{tx2, tx4}, txSet.SignedTransactions, "got transactions that failed pre-order validation")
}


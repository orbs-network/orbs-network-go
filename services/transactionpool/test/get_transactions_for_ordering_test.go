package test

import (
	"github.com/orbs-network/orbs-network-go/services/transactionpool"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGetTransactionsForOrderingDropsExpiredTransactions(t *testing.T) {
	t.Parallel()
	h := newHarness()

	validTx := builders.TransferTransaction().Build()
	expiredTx := builders.TransferTransaction().WithTimestamp(time.Now().Add(-1 * time.Duration(transactionExpirationWindow+60) * time.Second)).Build()

	// we use forward rather than add to simulate a scenario where a byzantine node submitted invalid transactions
	h.handleForwardFrom(otherNodeKeyPair, validTx, expiredTx)

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
	})

	txSet, err := h.getTransactionsForOrdering(4)

	require.NoError(t, err, "expected transaction set but got an error")
	require.ElementsMatch(t, transactionpool.Transactions{tx2, tx4}, txSet.SignedTransactions, "got transactions that failed pre-order validation")
}

func TestGetTransactionsForOrderingDropsTransactionsThatAreAlreadyCommitted(t *testing.T) {
	t.Parallel()
	h := newHarness()

	h.ignoringForwardMessages()

	tx1 := builders.TransferTransaction().Build()
	h.addTransactions(tx1)
	h.assumeBlockStorageAtHeight(1)
	h.ignoringTransactionResults()
	h.reportTransactionsAsCommitted(tx1) // this commits tx1, it will now be in the committed pool

	tx2 := builders.TransferTransaction().Build()

	h.handleForwardFrom(otherNodeKeyPair, tx1) // now we add the same transaction again as well as a new transaction
	h.addTransactions(tx2)

	txSet, err := h.getTransactionsForOrdering(2)

	require.NoError(t, err, "failed getting transactions unexpectedly")
	require.ElementsMatch(t, transactionpool.Transactions{tx2}, txSet.SignedTransactions, "got a transaction that has already been committed")

}

func TestGetTransactionsForOrderingRemovesCommittedTransactionsFromPool(t *testing.T) {
	t.Parallel()
	h := newHarness()

	h.ignoringForwardMessages()

	tx1 := builders.TransferTransaction().Build()
	h.addTransactions(tx1)
	h.assumeBlockStorageAtHeight(1)
	h.ignoringTransactionResults()
	h.reportTransactionsAsCommitted(tx1) // this commits tx1, it will now be in the committed pool

	tx2 := builders.TransferTransaction().Build()

	h.handleForwardFrom(otherNodeKeyPair, tx1) // now we add the same transaction again as well as a new transaction
	h.addTransactions(tx2)

	txSet, err := h.getTransactionsForOrdering(1)

	require.NoError(t, err, "failed getting transactions unexpectedly")
	require.Empty(t, txSet.SignedTransactions, "got a transaction that has already been committed")

	txSet, err = h.getTransactionsForOrdering(1)
	require.Len(t, txSet.SignedTransactions, 1, "did not get a valid transaction from the pool")
}

func TestGetTransactionsForOrderingRemovesTransactionsThatFailedPreOrderChecksFromPool(t *testing.T) {
	t.Parallel()
	h := newHarness()

	h.ignoringForwardMessages()

	tx1 := builders.TransferTransaction().Build()
	tx2 := builders.TransferTransaction().WithAmountAndTargetAddress(8, builders.AddressForEd25519SignerForTests(2)).Build()

	h.addTransactions(tx1, tx2)

	h.failPreOrderCheckFor(func(tx *protocol.SignedTransaction) bool {
		return tx == tx1
	})

	txSet, err := h.getTransactionsForOrdering(1)

	require.NoError(t, err, "failed getting transactions unexpectedly")
	require.Empty(t, txSet.SignedTransactions, "got a transaction that failed pre-order checks")

	txSet, _ = h.getTransactionsForOrdering(1)
	require.Len(t, txSet.SignedTransactions, 1, "did not get a valid transaction from the pool")
}

func TestGetTransactionsForOrderingRemovesInvalidTransactionsFromPool(t *testing.T) {
	t.Parallel()
	h := newHarness()

	expiredTx := builders.TransferTransaction().WithTimestamp(time.Now().Add(-1 * time.Duration(transactionExpirationWindow+60) * time.Second)).Build()
	validTx := builders.TransferTransaction().Build()

	// we use forward rather than add to simulate a scenario where a byzantine node submitted invalid transactions
	h.handleForwardFrom(otherNodeKeyPair, expiredTx, validTx)

	txSet, _ := h.getTransactionsForOrdering(1)
	require.Empty(t, txSet.SignedTransactions, "got an invalid transaction")

	txSet, _ = h.getTransactionsForOrdering(1)
	require.Len(t, txSet.SignedTransactions, 1, "did not get a valid transaction from the pool")

}

func TestGetTransactionsForOrderingAsOfFutureBlockHeightTimesOutWhenNoBlockIsCommitted(t *testing.T) {
	t.Parallel()
	h := newHarness()

	_, err := h.txpool.GetTransactionsForOrdering(&services.GetTransactionsForOrderingInput{
		BlockHeight:             2,
		MaxNumberOfTransactions: 1,
	})

	require.EqualError(t, err, "timed out waiting for block at height 2", "did not time out")
}

func TestGetTransactionsForOrderingAsOfFutureBlockHeightResolvesOutWhenBlockIsCommitted(t *testing.T) {
	t.Parallel()
	h := newHarness()

	doneWait := make(chan error)
	go func() {
		_, err := h.txpool.GetTransactionsForOrdering(&services.GetTransactionsForOrderingInput{
			BlockHeight:             1,
			MaxNumberOfTransactions: 1,
		})
		doneWait <- err
	}()

	h.assumeBlockStorageAtHeight(1)
	h.ignoringTransactionResults()
	h.reportTransactionsAsCommitted()

	require.NoError(t, <-doneWait, "did not resolve after block has been committed")
}

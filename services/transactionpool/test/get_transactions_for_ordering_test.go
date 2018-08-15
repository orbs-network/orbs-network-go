package test

import (
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
	expiredTx := builders.TransferTransaction().WithTimestamp(time.Now().Add(-1 * time.Duration(transactionExpirationWindowInSeconds+60) * time.Second)).Build()

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
	})

	txSet, err := h.getTransactionsForOrdering(4)

	require.NoError(t, err, "expected transaction set but got an error")
	require.ElementsMatch(t, []*protocol.SignedTransaction{tx2, tx4}, txSet.SignedTransactions, "got transactions that failed pre-order validation")
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

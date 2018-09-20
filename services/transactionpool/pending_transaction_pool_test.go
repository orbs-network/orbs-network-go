package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

var pk = keys.Ed25519KeyPairForTests(8).PublicKey()
var transactionExpirationWindow = 30 * time.Minute

func TestPendingTransactionPoolTracksSizesOfTransactionsAddedAndRemoved(t *testing.T) {
	t.Parallel()
	p := makePendingPool()
	require.Zero(t, p.currentSizeInBytes, "New pending pool created with non-zero size")

	tx1 := builders.TransferTransaction().Build()
	k1, _ := p.add(tx1, pk)
	require.Equal(t, uint32(len(tx1.Raw())), p.currentSizeInBytes, "pending pool size did not reflect tx1 size")

	tx2 := builders.TransferTransaction().WithContract("a contract with a long name so that tx has a different size").Build()
	k2, _ := p.add(tx2, pk)
	require.Equal(t, uint32(len(tx1.Raw())+len(tx2.Raw())), p.currentSizeInBytes, "pending pool size did not reflect combined sizes of tx1 + tx2")

	p.remove(k1)
	require.Equal(t, uint32(len(tx2.Raw())), p.currentSizeInBytes, "pending pool size did not reflect removal of tx1")

	p.remove(k2)
	require.Zero(t, p.currentSizeInBytes, "pending pool size did not reflect removal of tx2")
}

func TestPendingTransactionPoolAddRemoveKeepsBothDataStructuresInSync(t *testing.T) {
	t.Parallel()
	p := makePendingPool()
	tx1 := builders.TransferTransaction().Build()

	k, _ := p.add(tx1, pk)
	require.True(t, p.has(tx1), "has() returned false for an added item")
	require.Len(t, p.getBatch(1, 0), 1, "getBatch() did not return an added item")

	p.remove(k)
	require.False(t, p.has(tx1), "has() returned true for removed item")
	require.Empty(t, p.getBatch(1, 0), "getBatch() returned a removed item")

	require.NotPanics(t, func() {
		p.remove(k)
	}, "removing a key that does not exist resulted in a panic")
}

func TestPendingTransactionPoolGetBatchReturnsLessThanMaximumIfPoolHasLessTransaction(t *testing.T) {
	t.Parallel()
	p := makePendingPool()

	add(p, builders.TransferTransaction().Build(), builders.TransferTransaction().Build())

	txSet := p.getBatch(3, 0)

	require.Len(t, txSet, 2, "expected 2 transactions but got %v transactions: %s", len(txSet), txSet)
}

func TestPendingTransactionPoolGetBatchDoesNotExceedSizeLimitInBytes(t *testing.T) {
	t.Parallel()
	p := makePendingPool()

	tx1 := builders.TransferTransaction().Build()
	tx2 := builders.TransferTransaction().Build()
	add(p, tx1, tx2, builders.TransferTransaction().Build())

	slightlyMoreThanTwoTransactionsInBytes := uint32(len(tx1.Raw()) + len(tx2.Raw()) + 1)
	txSet := p.getBatch(3, slightlyMoreThanTwoTransactionsInBytes)

	require.Len(t, txSet, 2, "expected 2 transactions but got %v transactions: %s", len(txSet), txSet)
}

func TestPendingTransactionPoolGetBatchDoesNotExceedLengthLimit(t *testing.T) {
	t.Parallel()
	p := makePendingPool()

	tx1 := builders.TransferTransaction().Build()
	tx2 := builders.TransferTransaction().Build()
	tx3 := builders.TransferTransaction().Build()
	add(p, tx1, tx2, tx3)

	txSet := p.getBatch(2, 0)

	require.Len(t, txSet, 2, "expected 2 transactions but got %v transactions: %s", len(txSet), txSet)
}

func TestPendingTransactionPoolGetBatchRetainsInsertionOrder(t *testing.T) {
	t.Parallel()
	p := makePendingPool()

	// create 50 transactions so as to minimize the chance of randomly returning transactions in the expected order
	transactions := make(Transactions, 50, 50)
	for i := 0; i < len(transactions); i++ {
		transactions[i] = builders.TransferTransaction().Build()
		add(p, transactions[i])
	}

	txSet := p.getBatch(uint32(len(transactions)), 0)

	require.Equal(t, transactions, txSet, "got transactions in wrong order")
}

func TestPendingTransactionPoolClearsExpiredTransactions(t *testing.T) {
	t.Parallel()
	p := makePendingPool()

	tx1 := builders.TransferTransaction().WithTimestamp(time.Now().Add(-5 * time.Minute)).Build()
	tx2 := builders.TransferTransaction().WithTimestamp(time.Now().Add(-29 * time.Minute)).Build()
	tx3 := builders.TransferTransaction().WithTimestamp(time.Now().Add(-31 * time.Minute)).Build()
	add(p, tx1, tx2, tx3)

	p.clearTransactionsOlderThan(time.Now().Add(-30 * time.Minute))

	require.True(t, p.has(tx1), "cleared non-expired transaction")
	require.True(t, p.has(tx2), "cleared non-expired transaction")
	require.False(t, p.has(tx3), "did not clear expired transaction")
}

func TestPendingTransactionPoolDoesNotAddTheSameTransactionTwiceRegardlessOfPublicKey(t *testing.T) {
	p := makePendingPool()

	tx := builders.Transaction().Build()

	_, err := p.add(tx, pk)
	require.Nil(t, err, "got an unexpected error adding the first transaction")

	_, err = p.add(tx, pk)
	require.Equal(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING, err.TransactionStatus, "did not get expected status code")

	someOtherPk := keys.Ed25519KeyPairForTests(3).PublicKey()
	_, err = p.add(tx, someOtherPk)
	require.Equal(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING, err.TransactionStatus, "did not get expected status code")

}

func add(p *pendingTxPool, txs ...*protocol.SignedTransaction) {
	for _, tx := range txs {
		p.add(tx, pk)
	}
}

func makePendingPool() *pendingTxPool {
	return NewPendingPool(func() uint32 { return 100000 })
}

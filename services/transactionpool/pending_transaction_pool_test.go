package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

var pk = keys.Ed25519KeyPairForTests(8).PublicKey()

func TestPendingTransactionPool_TracksSizesOfTransactionsAddedAndRemoved(t *testing.T) {
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

func TestPendingTransactionPool_AddRemove(t *testing.T) {
	t.Parallel()
	p := makePendingPool()
	tx1 := builders.TransferTransaction().Build()

	k, _ := p.add(tx1, pk)
	require.True(t, p.has(tx1), "pending pool did not add tx1")

	p.remove(k)
	require.False(t, p.has(tx1), "pending pool did not remove tx1")

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

func TestPendingTransactionPoolGetBatchDoesNotExceedSizeLimit(t *testing.T) {
	t.Parallel()
	p := makePendingPool()

	tx1 := builders.TransferTransaction().Build()
	tx2 := builders.TransferTransaction().Build()
	add(p, tx1, tx2, builders.TransferTransaction().Build())

	slightlyMoreThanTwoTransactionsInBytes := uint32(len(tx1.Raw()) + len(tx2.Raw()) + 1)
	txSet := p.getBatch(3, slightlyMoreThanTwoTransactionsInBytes)

	require.Len(t, txSet, 2, "expected 2 transactions but got %v transactions: %s", len(txSet), txSet)
}

func TestPendingTransactionPoolGetBatchDoesNotExceedLimitAndRetainsOrdering(t *testing.T) {
	t.Parallel()
	p := makePendingPool()

	now := time.Now()
	tx1 := builders.TransferTransaction().WithTimestamp(now.Add(-3 * time.Second)).Build()
	tx2 := builders.TransferTransaction().WithTimestamp(now.Add(-2 * time.Second)).Build()
	tx3 := builders.TransferTransaction().WithTimestamp(now.Add(-4 * time.Second)).Build()
	add(p, tx1, tx2, tx3)

	txSet := p.getBatch(2, 0)

	require.Len(t, txSet, 2, "expected 2 transactions but got %v transactions: %s", len(txSet), txSet)
	//require.Equal(t, []*protocol.SignedTransaction{tx3, tx1}, txSet, "got transaction set in wrong order") TODO re-enable when we decide on ordering (timestamp vs arrival)
}

//TODO size of transaction set does not exceed limit

func add(p *pendingTxPool, txs ...*protocol.SignedTransaction) {
	for _, tx := range txs {
		p.add(tx, pk)
	}
}

func makePendingPool() *pendingTxPool {
	return NewPendingPool(config.NewTransactionPoolConfig(100000, pk))
}

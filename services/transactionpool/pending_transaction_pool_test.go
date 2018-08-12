package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/stretchr/testify/require"
	"testing"
)

var pk = keys.Ed25519KeyPairForTests(8).PublicKey()

func TestPendingTransactionPool_TracksSizesOfTransactionsAddedAndRemoved(t *testing.T) {
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

func makePendingPool() *pendingTxPool {
	return NewPendingPool(config.NewTransactionPoolConfig(100000, pk))
}

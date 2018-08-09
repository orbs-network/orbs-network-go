package transactionpool

import (
	"testing"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/stretchr/testify/require"
	"github.com/orbs-network/orbs-network-go/test/builders"
	)

func TestPendingTransactionPool_TracksSizesOfTransactionsAddedAndRemoved(t *testing.T) {
	p := NewPendingPool(config.NewTransactionPoolConfig(100000))
	require.Zero(t, p.currentSizeInBytes, "New pending pool created with non-zero size")

	tx1 := builders.TransferTransaction().Build()
	k1, _ := p.add(tx1)
	require.Equal(t, uint32(len(tx1.Raw())), p.currentSizeInBytes, "Pending pool size did not reflect tx1 size")

	tx2 := builders.TransferTransaction().WithContract("a contract with a long name so that tx has a different size").Build()
	k2, _ := p.add(tx2)
	require.Equal(t, uint32(len(tx1.Raw()) + len(tx2.Raw())), p.currentSizeInBytes, "Pending pool size did not reflect combined sizes of tx1 + tx2")

	p.remove(k1)
	require.Equal(t, uint32(len(tx2.Raw())), p.currentSizeInBytes, "Pending pool size did not reflect removal of tx1")

	p.remove(k2)
	require.Zero(t, p.currentSizeInBytes, "Pending pool size did not reflect removal of tx2")
}

package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInterNodeBlockSync(t *testing.T) {
	harness.Network(t).
		AllowingErrors(
			"leader failed to save block to storage",                 // (block already in storage, skipping) TODO investigate and explain, or fix and remove expected error
			"all consensus \\d* algos refused to validate the block", //TODO investigate and explain, or fix and remove expected error
		).
		WithSetup(func(ctx context.Context, network harness.TestNetworkDriver) {
			for i := 1; i <= 10; i++ {
				blockPair := builders.BenchmarkConsensusBlockPair().
					WithHeight(primitives.BlockHeight(i)).
					WithTransactions(2).
					Build()
				network.BlockPersistence(0).WriteNextBlock(blockPair)
			}

			numBlocks, err := network.BlockPersistence(1).GetNumBlocks()
			require.NoError(t, err)
			require.Zero(t, numBlocks)
		}).Start(func(ctx context.Context, network harness.TestNetworkDriver) {

		err := network.BlockPersistence(0).GetBlockTracker().WaitForBlock(ctx, 10)
		require.NoError(t, err, "sanity wait on node 0 failed")

		err = network.BlockPersistence(1).GetBlockTracker().WaitForBlock(ctx, 5)
		require.NoError(t, err, "waiting for half sync on node 1 failed")

		// Wait until full sync
		err = network.BlockPersistence(1).GetBlockTracker().WaitForBlock(ctx, 10)
		require.NoError(t, err, "waiting for full sync on node 1 failed")

		// Wait again to get new blocks created after the sync
		err = network.BlockPersistence(1).GetBlockTracker().WaitForBlock(ctx, 15)
		require.NoError(t, err, "waiting for extra new blocks on node 1 failed")
	})
}

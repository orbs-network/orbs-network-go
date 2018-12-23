package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInterNodeBlockSync(t *testing.T) {

	harness.Network(t).
		//WithLogFilters(log.ExcludeEntryPoint("BenchmarkConsensus.Tick")).
		AllowingErrors(
			"leader failed to save block to storage",                 // (block already in storage, skipping) TODO(v1) investigate and explain, or fix and remove expected error
			"all consensus \\d* algos refused to validate the block", //TODO(v1) investigate and explain, or fix and remove expected error
		).
		WithSetup(func(ctx context.Context, network harness.TestNetworkDriver) {
			var prevBlock *protocol.BlockPairContainer
			for i := 1; i <= 10; i++ {
				blockPair := builders.BenchmarkConsensusBlockPair().
					WithHeight(primitives.BlockHeight(i)).
					WithTransactions(2).
					WithPrevBlock(prevBlock).
					Build()
				network.BlockPersistence(0).WriteNextBlock(blockPair)
				prevBlock = blockPair
			}

			numBlocks, err := network.BlockPersistence(1).GetLastBlockHeight()
			require.NoError(t, err)
			require.Zero(t, numBlocks)
		}).Start(func(ctx context.Context, network harness.TestNetworkDriver) {
		if err := network.BlockPersistence(0).GetBlockTracker().WaitForBlock(ctx, 10); err != nil {
			t.Errorf("waiting for block on node 0 failed: %s", err)
		}

		if err := network.BlockPersistence(1).GetBlockTracker().WaitForBlock(ctx, 5); err != nil {
			t.Errorf("waiting for block on node 1 failed: %s", err)
		}

		// Wait until full sync
		if err := network.BlockPersistence(1).GetBlockTracker().WaitForBlock(ctx, 10); err != nil {
			t.Errorf("waiting for block on node 1 failed: %s", err)
		}

		// Wait again to get new blocks created after the sync
		if err := network.BlockPersistence(1).GetBlockTracker().WaitForBlock(ctx, 15); err != nil {
			t.Errorf("waiting for block on node 1 failed: %s", err)
		}
	})
}

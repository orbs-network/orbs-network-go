package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBlockSync(t *testing.T) {
	harness.Network(t).
		AllowingErrors(
			"leader failed to save block to storage",              // (block already in storage, skipping) TODO investigate and explain, or fix and remove expected error
			"intra-node sync to consensus algo failed",            //TODO investigate and explain, or fix and remove expected error
			"all consensus 0 algos refused to validate the block", //TODO investigate and explain, or fix and remove expected error
			"all consensus 1 algos refused to validate the block", //TODO investigate and explain, or fix and remove expected error
		).
		WithSetup(func(ctx context.Context, network harness.InProcessTestNetwork) {
			for i := 1; i <= 10; i++ {
				blockPair := builders.BenchmarkConsensusBlockPair().
					WithHeight(primitives.BlockHeight(i)).
					WithTransactions(2).
					Build()
				network.BlockPersistence(0).WriteBlock(blockPair)

			}
		}).Start(func(ctx context.Context, network harness.InProcessTestNetwork) {
		require.Zero(t, len(network.BlockPersistence(1).ReadAllBlocks()))

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

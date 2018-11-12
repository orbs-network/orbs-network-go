package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"testing"
	"time"
)

func TestIntraBlockSync(t *testing.T) {
	harness.Network(t).
		AllowingErrors(
			"leader failed to save block to storage",              // (block already in storage, skipping) TODO investigate and explain, or fix and remove expected error
			"internal-node sync to consensus algo failed",            //TODO investigate and explain, or fix and remove expected error
			"all consensus 0 algos refused to validate the block", //TODO investigate and explain, or fix and remove expected error
			"all consensus 1 algos refused to validate the block", //TODO investigate and explain, or fix and remove expected error
		).
		WithSetup(func(ctx context.Context, network harness.TestNetworkDriver) {
			for i := 1; i <= 10; i++ {
				blockPair := builders.BenchmarkConsensusBlockPair().
					WithHeight(primitives.BlockHeight(i)).
					WithTransactions(2).
					Build()
				network.BlockPersistence(0).WriteNextBlock(blockPair)

			}
		}).Start(func(ctx context.Context, network harness.TestNetworkDriver) {
		ctx, _ = context.WithTimeout(ctx, 1 * time.Second)

		// Wait for state storage to sync to block height 15
		if err := network.BlockPersistence(0).GetBlockTracker().WaitForBlock(ctx, 10); err != nil {
			t.Errorf("waiting for block on node 0 failed: %s", err)
		}

		// Wait for state storage to sync to block height 15
		if err := network.StatePersistence(0).WaitUntilCommittedBlockOfHeight(ctx, 10); err != nil {
			t.Errorf("waiting for state storage to sync on node 0 failed: %s", err)
		}
	})
}

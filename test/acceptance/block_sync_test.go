package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"testing"
)

func TestBlockSync(t *testing.T) {
	t.Skip("hmm")

	harness.Network(t).WithSetup(func(ctx context.Context, network harness.InProcessNetwork) {
		for i := 1; i <= 10; i++ {
			blockPair := builders.BlockPair().WithHeight(primitives.BlockHeight(i)).Build()
			network.BlockPersistence(0).WriteBlock(blockPair)
		}
	}).Start(func(ctx context.Context, network harness.InProcessNetwork) {
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

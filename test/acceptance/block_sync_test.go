package acceptance

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestBlockSync(t *testing.T) {
	harness.Network(t).WithSetup(func(network harness.InProcessNetwork) {
		for i := 1; i <= 10; i++ {
			blockPair := builders.BlockPair().WithHeight(primitives.BlockHeight(i)).WithTransactions(2).Build()
			network.BlockPersistence(0).WriteBlock(blockPair)
		}
	}).Start(func(network harness.InProcessNetwork) {
		require.Zero(t, len(network.BlockPersistence(1).ReadAllBlocks()))

		if err := network.BlockPersistence(0).GetBlockTracker().WaitForBlock(10); err != nil {
			t.Errorf("waiting for block on node 0 failed: %s", err)
		}

		if err := network.BlockPersistence(1).GetBlockTracker().WaitForBlock(5); err != nil {
			t.Errorf("waiting for block on node 1 failed: %s", err)
		}

		time.Sleep(1 * time.Second)

		//// Wait until full sync
		if err := network.BlockPersistence(1).GetBlockTracker().WaitForBlock(10); err != nil {
			t.Errorf("waiting for block on node 1 failed: %s", err)
		}

		// Wait again to get new blocks created after the sync
		if err := network.BlockPersistence(1).GetBlockTracker().WaitForBlock(15); err != nil {
			t.Errorf("waiting for block on node 1 failed: %s", err)
		}
	})
}

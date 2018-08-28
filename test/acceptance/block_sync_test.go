package acceptance

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"testing"
)

func TestBlockSync(t *testing.T) {
	harness.Network(t).WithSetup(func(network harness.AcceptanceTestNetwork) {
		for i := 1; i <= 10; i++ {
			blockPair := builders.BlockPair().WithHeight(primitives.BlockHeight(i)).Build()
			network.BlockPersistence(0).WriteBlock(blockPair)
		}
	}).Start(func(network harness.AcceptanceTestNetwork) {
		if err := network.BlockPersistence(0).GetBlockTracker().WaitForBlock(10); err != nil {
			t.Errorf("waiting for block on node 0 failed: %s", err)
		}

		if err := network.BlockPersistence(1).GetBlockTracker().WaitForBlock(5); err != nil {
			t.Errorf("waiting for block on node 1 failed: %s", err)
		}

		if err := network.BlockPersistence(1).GetBlockTracker().WaitForBlock(10); err != nil {
			t.Errorf("waiting for block on node 1 failed: %s", err)
		}

		// TODO change the test to check that consensus keeps producing new blocks
		//if err := network.BlockPersistence(1).GetBlockTracker().WaitForBlock(10); err != nil {
		//	t.Errorf("waiting for block on node 1 failed: %s", err)
		//}
	})
}

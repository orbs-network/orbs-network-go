package acceptance

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"math/rand"
	"strconv"
	"testing"
)

func TestBlockSync(t *testing.T) {
	testId := "acceptance-BlockSync-" + strconv.FormatUint(rand.Uint64(), 10)
	defer harness.ReportTestId(t, testId)

	harness.WithNetwork(t, testId, 2, harness.WithAlgos(consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS), func(network harness.AcceptanceTestNetwork) {
		for i := 1; i < 5; i++ {
			blockPair := builders.BlockPair().WithHeight(primitives.BlockHeight(i))
			network.BlockPersistence(0).WriteBlock(blockPair.Build())
		}

		if err := network.BlockPersistence(0).GetBlockTracker().WaitForBlock(5); err != nil {
			t.Errorf("waiting for block on node 0 failed: %s", err)
		}
		//require.EqualValues(t, 17, <-network.CallGetBalance(0), "eventual getBalance result on leader")

		if err := network.BlockPersistence(1).GetBlockTracker().WaitForBlock(5); err != nil {
			t.Errorf("waiting for block on node 1 failed: %s", err)
		}
		//require.EqualValues(t, 17, <-network.CallGetBalance(1), "eventual getBalance result on non leader")

	})
}

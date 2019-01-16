package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNetworkStartedWithEnoughNodes_SucceedsClosingBlocks_BenchmarkConsensus(t *testing.T) {
	harness.Network(t).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS).
		WithNumNodes(6).
		WithNumRunningNodes(4).
		WithRequiredQuorumPercentage(66). // this is used only by benchmark consensus
		WithLogFilters(log.ExcludeEntryPoint("BlockSync")).
		Start(func(parent context.Context, network harness.TestNetworkDriver) {
			ctx, cancel := context.WithTimeout(parent, 1*time.Second)
			defer cancel()

			contract := network.BenchmarkTokenContract()
			contract.DeployBenchmarkToken(ctx, 5)

			out, _ := contract.Transfer(ctx, 0, uint64(23), 5, 6)
			require.NotNil(t, out)
		})
}

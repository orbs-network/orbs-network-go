package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNetworkStartedWithEnoughNodes_SucceedsClosingBlocks_BenchmarkConsensus(t *testing.T) {
	newHarness(t).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS).
		WithNumNodes(6).
		WithNumRunningNodes(4).
		WithRequiredQuorumPercentage(66).
		WithLogFilters(
			log.ExcludeEntryPoint("BlockSync"),
			log.IgnoreMessagesMatching("Metric recorded"),
			log.ExcludeEntryPoint("LeanHelixConsensus")).
		Start(func(ctx context.Context, network NetworkHarness) {
			contract := network.DeployBenchmarkTokenContract(ctx, 5)

			out, _ := contract.Transfer(ctx, 0, uint64(23), 5, 6)
			require.NotNil(t, out)
		})
}

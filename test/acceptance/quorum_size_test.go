package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNetworkStartedWithEnoughNodes_SucceedsClosingBlocks(t *testing.T) {
	harness.Network(t).
		WithNumNodes(6).
		WithNumRunningNodes(4).
		WithRequiredQuorumPercentage(66).
		WithLogFilters(
			log.ExcludeEntryPoint("BlockSync"),
			log.IgnoreMessagesMatching("Metric recorded"),
			log.ExcludeEntryPoint("LeanHelixConsensus")).
		Start(func(ctx context.Context, network harness.TestNetworkDriver) {
			contract := network.BenchmarkTokenContract()
			contract.DeployBenchmarkToken(ctx, 5)

			out, _ := contract.Transfer(ctx, 0, uint64(23), 5, 6)
			require.NotNil(t, out)
		})
}

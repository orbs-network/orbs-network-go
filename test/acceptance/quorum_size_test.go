package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNetworkStartedWithEnoughNodes_SucceedsClosingBlocks(t *testing.T) {
	harness.Network(t).
		WithNumNodes(6).
		WithNumRunningNodes(4).
		WithRequiredQuorumPercentage(66).
		WithLogFilters(log.ExcludeEntryPoint("BlockSync")).
		Start(func(parent context.Context, network harness.TestNetworkDriver) {
			ctx, cancel := context.WithTimeout(parent, 1*time.Second)
			defer cancel()

			contract := network.GetBenchmarkTokenContract()
			contract.DeployBenchmarkToken(ctx, 5)

			require.NotNil(t, contract.SendTransfer(ctx, 0, uint64(23), 5, 6))
		})
}

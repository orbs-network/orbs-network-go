package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
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

			requireSuccess(t, contract.SendTransfer(ctx, 0, uint64(23), 5, 6))
		})
}

func requireSuccess(t *testing.T, ch chan *client.SendTransactionResponse) {
	select {
	case res := <-ch:
		test.RequireSuccess(t, res, "transaction should be successfully committed and executed")
	case <-time.After(2000 * time.Millisecond):
		t.Fatalf("transaction did not succeed within timeout")
	}
}

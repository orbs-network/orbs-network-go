package development

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// Gamma is based on harness.NewDevelopmentNetwork instead of harness.NewAcceptanceTestNetwork
// NewDevelopmentNetwork is almost identical to NewAcceptanceTestNetwork (in-memory adapters) except it uses real compilation (real processor/native/adapter)
// this test is very similar to the acceptance test, just checks contract deployment with real compilation
func TestNonLeaderDeploysNativeContract(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping compilation of contracts in short mode")
	}

	test.WithContext(func(ctx context.Context) {
		network := harness.NewDevelopmentNetwork(log.GetLogger()).Start(ctx)

		counterStart := contracts.MOCK_COUNTER_CONTRACT_START_FROM

		t.Log("deploying contract")

		<-network.GetCounterContract().SendDeployCounterContract(ctx, 1) // leader is nodeIndex 0, validator is nodeIndex 1
		require.EqualValues(t, counterStart, <-network.GetCounterContract().CallCounterGet(ctx, 0), "get counter after deploy")

		t.Log("transacting with contract")

		<-network.GetCounterContract().SendCounterAdd(ctx, 1, 17)
		require.EqualValues(t, counterStart+17, <-network.GetCounterContract().CallCounterGet(ctx, 0), "get counter after transaction")

	})
	time.Sleep(5 * time.Millisecond) // give context dependent goroutines 5 ms to terminate gracefully
}

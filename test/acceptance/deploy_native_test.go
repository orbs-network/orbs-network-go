package acceptance

import (
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/stretchr/testify/require"
	"math/rand"
	"testing"
)

func TestNonLeaderDeploysNativeContract(t *testing.T) {
	t.Skip("Too slow for acceptance test (but working)")
	harness.Network(t).Start(func(network harness.AcceptanceTestNetwork) {

		t.Log("testing", network.Description()) // leader is nodeIndex 0, validator is nodeIndex 1

		counterStart := uint64(1000 * rand.Intn(100))

		t.Log("deploying contract")

		<-network.SendDeployCounterContract(1, counterStart)
		require.EqualValues(t, counterStart, <-network.CallCounterGet(0, counterStart), "get counter after deploy")

		t.Log("transacting with contract")

		<-network.SendCounterAdd(1, counterStart, 17)
		require.EqualValues(t, counterStart+17, <-network.CallCounterGet(0, counterStart), "get counter after transaction")

	})
}

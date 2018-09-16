package acceptance

import (
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNonLeaderDeploysNativeContract(t *testing.T) {
	harness.Network(t).Start(func(network harness.AcceptanceTestNetwork) {

		t.Log("testing", network.Description()) // leader is nodeIndex 0, validator is nodeIndex 1

		counterStart := contracts.MOCK_COUNTER_CONTRACT_START_FROM

		t.Log("deploying contract")

		<-network.SendDeployCounterContract(1)
		require.EqualValues(t, counterStart, <-network.CallCounterGet(0), "get counter after deploy")

		t.Log("transacting with contract")

		<-network.SendCounterAdd(1, 17)
		require.EqualValues(t, counterStart+17, <-network.CallCounterGet(0), "get counter after transaction")

	})
}

package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/acceptance/callcontract"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/stretchr/testify/require"
	"testing"
)

// LH: Use ControlledRandom (ctrlrnd.go) (in acceptance harness) to generate the initial RandomSeed and put it in LeanHelix's config (remove "NonLeader")
func TestDeploysNativeContract(t *testing.T) {
	newHarness().Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {

		// in BC leader is nodeIndex 0, validator is nodeIndex 1, in LH leadership is randomized

		counterStart := contracts.MOCK_COUNTER_CONTRACT_START_FROM
		network.MockContract(contracts.MockForCounter(), string(contracts.NativeSourceCodeForCounter(counterStart)))
		contract := callcontract.NewContractClient(network)

		t.Log("deploying contract")

		_, txHash := contract.DeployCounterContract(ctx, 1)

		t.Log("wait for node to sync with deployment")
		network.WaitForTransactionInNodeState(ctx, txHash, 0)

		require.EqualValues(t, counterStart, contract.CounterGet(ctx, 0), "get counter after deploy")

		t.Log("transacting with contract")

		_, txHash = contract.CounterAdd(ctx, 1, 17)

		t.Log("wait for node to sync with transaction")
		network.WaitForTransactionInNodeState(ctx, txHash, 0)

		require.EqualValues(t, counterStart+17, contract.CounterGet(ctx, 0), "get counter after transaction")

	})
}

package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/orbs-network/orbs-network-go/test/harness"
	contractClient "github.com/orbs-network/orbs-network-go/test/harness/contracts"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNonLeaderDeploysNativeContract(t *testing.T) {
	harness.Network(t).Start(func(ctx context.Context, network harness.TestNetworkDriver) {

		t.Log("testing", network.Description()) // leader is nodeIndex 0, validator is nodeIndex 1

		counterStart := contracts.MOCK_COUNTER_CONTRACT_START_FROM
		network.MockContract(contracts.MockForCounter(), string(contracts.NativeSourceCodeForCounter(counterStart)))
		contract := contractClient.NewContractClient(network)

		t.Log("deploying contract")

		_, txHash := contract.SendDeployCounterContract(ctx, 1)

		t.Log("wait for node to sync with deployment")
		network.WaitForTransactionInNodeState(ctx, txHash, 0)

		require.EqualValues(t, counterStart, contract.CallCounterGet(ctx, 0), "get counter after deploy")

		t.Log("transacting with contract")

		_, txHash = contract.SendCounterAdd(ctx, 1, 17)

		t.Log("wait for node to sync with transaction")
		network.WaitForTransactionInNodeState(ctx, txHash, 0)

		require.EqualValues(t, counterStart+17, contract.CallCounterGet(ctx, 0), "get counter after transaction")

	})
}

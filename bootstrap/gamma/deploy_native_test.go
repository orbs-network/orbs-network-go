package gamma

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	contractClient "github.com/orbs-network/orbs-network-go/test/harness/contracts"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNonLeaderDeploysNativeContract(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping compilation of contracts in short mode")
	}

	test.WithContext(func(ctx context.Context) {
		network := NewDevelopmentNetwork(ctx, log.GetLogger())
		contract := contractClient.NewContractClient(network)

		counterStart := contracts.MOCK_COUNTER_CONTRACT_START_FROM

		t.Log("deploying contract")

		output := contract.SendDeployCounterContract(ctx, 1)                         // leader is nodeIndex 0, validator is nodeIndex 1
		network.WaitForTransactionInState(ctx, output.TransactionReceipt().Txhash()) // wait for contract deployment take effect in node state

		require.EqualValues(t, counterStart, contract.CallCounterGet(ctx, 0), "get counter after deploy")

		t.Log("transacting with contract")

		output = contract.SendCounterAdd(ctx, 1, 17)
		network.WaitForTransactionInState(ctx, output.TransactionReceipt().Txhash())

		require.EqualValues(t, counterStart+17, contract.CallCounterGet(ctx, 0), "get counter after transaction")

	})
	time.Sleep(5 * time.Millisecond) // give context dependent goroutines 5 ms to terminate gracefully
}

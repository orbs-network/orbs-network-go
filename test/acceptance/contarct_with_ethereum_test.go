package acceptance

import (
	"context"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/internodesync"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/contracts/ethereum_caller"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDeployAndCallContractThatCallsEthereum(t *testing.T) {
	harness.Network(t).
		WithLogFilters(log.ExcludeField(internodesync.LogTag), log.ExcludeEntryPoint("tx-pool-sync")).
		Start(func(ctx context.Context, network harness.TestNetworkDriver) {

			addressOfContractInEthereum, err := network.EthereumSimulator().DeployStorageContract(ctx, 0, "foobar")
			require.NoError(t, err, "deploy of storage contract failed")

			test.RequireSuccess(t, deployOrbsContractCallingEthereum(ctx, network), "failed deploying the EthereumReader contract")

			readResponse := readStringFromEthereumReaderAt(ctx, network, addressOfContractInEthereum)

			require.EqualValues(t, protocol.EXECUTION_RESULT_SUCCESS, readResponse.CallMethodResult())
			require.EqualValues(t, "foobar", extractStringValueFrom(readResponse))

		})
}

func extractStringValueFrom(readResponse *client.CallMethodResponse) string {
	return builders.ClientCallMethodResponseOutputArgumentsDecode(readResponse).NextArguments().StringValue()
}

func readStringFromEthereumReaderAt(ctx context.Context, network harness.TestNetworkDriver, address string) *client.CallMethodResponse {
	readTx := builders.Transaction().
		WithMethod("EthereumReader", "readString").
		WithArgs(address).
		Builder()
	readResponse := <-network.CallMethod(ctx, readTx.Transaction, 0)
	return readResponse
}

func deployOrbsContractCallingEthereum(ctx context.Context, network harness.TestNetworkDriver) *client.SendTransactionResponse {
	ethereumReaderCode := "foo"
	network.MockContract(&sdkContext.ContractInfo{
		PublicMethods: ethereum_caller.PUBLIC,
		SystemMethods: ethereum_caller.SYSTEM,
		Permission:    sdkContext.PERMISSION_SCOPE_SERVICE,
	}, ethereumReaderCode)
	deployTx := builders.Transaction().
		WithMethod("_Deployments", "deployService").
		WithArgs(
			"EthereumReader",
			uint32(protocol.PROCESSOR_TYPE_NATIVE),
			[]byte(ethereumReaderCode),
		).Builder()

	return <-network.SendTransaction(ctx, deployTx, 0)
}

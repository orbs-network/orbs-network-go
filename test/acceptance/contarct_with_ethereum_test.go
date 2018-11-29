package acceptance

import (
	"context"
	"github.com/ethereum/go-ethereum/common/hexutil"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/internodesync"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/contracts/ethereum_caller"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestDeployAndCallContractThatCallsEthereum(t *testing.T) {
	harness.Network(t).
		WithLogFilters(log.ExcludeField(internodesync.LogTag), log.ExcludeEntryPoint("tx-pool-sync"), log.ExcludeEntryPoint("TransactionForwarder")).
		Start(func(ctx context.Context, network harness.TestNetworkDriver) {

		addressOfContractInEthereum := deployEthereumContract(t, network.EthereumSimulator(), "foobar")
		deployOrbsContractCallingEthereum(ctx, network)

		require.NoError(t, ctx.Err(), "failed deploying the EthereumReader contract")

		readResponse := readStringFromEthereumReaderAt(ctx, network, addressOfContractInEthereum)

		require.EqualValues(t, protocol.EXECUTION_RESULT_SUCCESS, readResponse.CallMethodResult())
		require.EqualValues(t, "foobar", extractStringValueFrom(readResponse))

	})
}

func deployEthereumContract(t *testing.T, simulator *adapter.EthereumSimulator, stringValue string) string {
	addressOfContractInEthereum, err := simulator.DeploySimpleStorageContract(simulator.GetAuth(), stringValue)
	simulator.Commit()
	require.NoError(t, err, "deploy of storage contract failed")
	return hexutil.Encode(addressOfContractInEthereum[:])
}

func extractStringValueFrom(readResponse *client.CallMethodResponse) string {
	return builders.ClientCallMethodResponseOutputArgumentsDecode(readResponse).NextArguments().StringValue()
}

func readStringFromEthereumReaderAt(ctx context.Context, network harness.TestNetworkDriver, address string) *client.CallMethodResponse {
	readTx := builders.Transaction().
		WithMethod("EthereumReader", "readString").
		WithArgs(address).
		Builder()
	readResponse := network.CallMethod(ctx, readTx.Transaction, 0)
	return readResponse
}

func deployOrbsContractCallingEthereum(parent context.Context, network harness.TestNetworkDriver)  {
	ctx, cancel := context.WithTimeout(parent, 2 * time.Second)
	defer cancel()
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

	network.SendTransactionInBackground(ctx, deployTx, 0)
	network.WaitForTransactionInState(ctx, digest.CalcTxHash(deployTx.Build().Transaction()))
}

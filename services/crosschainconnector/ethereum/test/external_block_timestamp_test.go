package test

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/contract"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/stretchr/testify/require"
	"math/big"
	"strings"
	"testing"
	"time"
)

//TODO this test does not deal well with gaps in ganache, meaning that a ganache that has been standing idle for more than a few seconds will fail this test while
// trying to assert that a contract cannot be called before it has been deployed
func TestFullFlowWithVaryingTimestamps(t *testing.T) {
	// the idea of this test is to make sure that the entire 'call-from-ethereum' logic works on a specific timestamp and different states in time (blocks)
	// it requires ganache or some other RPC backend to transact

	if !runningWithDocker() {
		t.Skip("this test relies on external components - ganache, and will be skipped unless running in docker")
	}

	test.WithContext(func(ctx context.Context) {
		h := newRpcEthereumConnectorHarness(t, ConfigForExternalRPCConnection())
		latestBlockInGanache, err := h.rpcAdapter.HeaderByNumber(ctx, nil)
		require.NoError(t, err, "failed to get latest block in ganache")

		timeBeforeContractWasDeployed := time.Unix(latestBlockInGanache.Time.Int64(), 0)
		h.moveBlocksInGanache(t, 2, 1) // this is only to advance blocks

		expectedTextFromEthereum := "test3"
		contractAddress3, err := h.deployRpcStorageContract(expectedTextFromEthereum)
		require.NoError(t, err, "failed deploying contract3 to Ethereum")

		h.moveBlocksInGanache(t, 11, 1) // this is only to advance blocks

		methodToCall := "getValues"

		parsedABI, err := abi.JSON(strings.NewReader(contract.SimpleStorageABI))
		require.NoError(t, err, "abi parse failed for simple storage contract")

		ethCallData, err := ethereum.ABIPackFunctionInputArguments(parsedABI, methodToCall, nil)
		require.NoError(t, err, "this means we couldn't pack the params for ethereum, something is broken with the harness")

		input := builders.EthereumCallContractInput().
			WithTimestamp(timeBeforeContractWasDeployed.Add(14 * time.Second)).
			WithContractAddress(contractAddress3).
			WithAbi(contract.SimpleStorageABI).
			WithFunctionName(methodToCall).
			WithPackedArguments(ethCallData).
			Build()

		output, err := h.connector.EthereumCallContract(ctx, input)
		require.NoError(t, err, "expecting call to succeed")
		require.True(t, len(output.EthereumAbiPackedOutput) > 0, "expecting output to have some data")
		ret := new(struct { // this is the expected return type of that ethereum call for the SimpleStorage contract getValues
			IntValue    *big.Int
			StringValue string
		})

		ethereum.ABIUnpackFunctionOutputArguments(parsedABI, ret, methodToCall, output.EthereumAbiPackedOutput)
		require.Equal(t, expectedTextFromEthereum, ret.StringValue, "text part from eth")

		input = builders.EthereumCallContractInput().
			WithTimestamp(timeBeforeContractWasDeployed).
			WithContractAddress(contractAddress3).
			WithAbi(contract.SimpleStorageABI).
			WithFunctionName(methodToCall).
			WithPackedArguments(ethCallData).
			Build()

		output, err = h.connector.EthereumCallContract(ctx, input)
		require.Error(t, err, "expecting call to fail as contract is not yet deployed in a past time block")
	})
}

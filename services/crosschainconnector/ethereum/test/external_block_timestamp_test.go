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
	"os"
	"strings"
	"testing"
	"time"
)

func TestFullFlowWithVaryingTimestamps(t *testing.T) {
	// the idea of this test is to make sure that the entire 'call-from-ethereum' logic works on a specific timestamp and different states in time (blocks)
	// it requires ganache or some other simulation to transact

	if !runningWithDocker() {
		t.Skip("this test relies on external components - ganache, and will be skipped unless running in docker")
	}

	test.WithContext(func(ctx context.Context) {
		h := newRpcEthereumConnectorHarness(t, getConfig())
		h.deployContractsToGanache(t, 2, time.Second)

		expectedTextFromEthereum := "test3"
		contractAddress3, err := h.deployRpcStorageContract(expectedTextFromEthereum)
		require.NoError(t, err, "failed deploying contract3 to Ethereum")

		methodToCall := "getValues"

		parsedABI, err := abi.JSON(strings.NewReader(contract.SimpleStorageABI))
		require.NoError(t, err, "abi parse failed for simple storage contract")

		ethCallData, err := ethereum.ABIPackFunctionInputArguments(parsedABI, methodToCall, nil)
		require.NoError(t, err, "this means we couldn't pack the params for ethereum, something is broken with the harness")

		input := builders.EthereumCallContractInput().
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
			WithTimestamp(time.Now().Add(time.Duration(-3) * time.Second)).
			WithContractAddress(contractAddress3).
			WithAbi(contract.SimpleStorageABI).
			WithFunctionName(methodToCall).
			WithPackedArguments(ethCallData).
			Build()

		output, err = h.connector.EthereumCallContract(ctx, input)
		require.Error(t, err, "expecting call to fail as contract is not yet deployed in a past time block")
	})
}

func runningWithDocker() bool {
	return os.Getenv("EXTERNAL_TEST") == "true"
}

func getConfig() *ethereumConnectorConfigForTests {
	var cfg ethereumConnectorConfigForTests

	if endpoint := os.Getenv("ETHEREUM_ENDPOINT"); endpoint != "" {
		cfg.endpoint = endpoint
	}

	if privateKey := os.Getenv("ETHEREUM_PRIVATE_KEY"); privateKey != "" {
		cfg.privateKeyHex = privateKey
	}

	return &cfg
}

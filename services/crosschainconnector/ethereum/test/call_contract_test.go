package test

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
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

func TestContractCallBadNodeConfig(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
		config := &ethereumConnectorConfigForTests{"all your base", ""}
		conn := adapter.NewEthereumRpcConnection(config, logger)
		connector := ethereum.NewEthereumCrosschainConnector(conn, logger)
		input := builders.EthereumCallContractInput().Build() // don't care about specifics

		_, err := connector.EthereumCallContract(ctx, input)
		require.Error(t, err, "expected call to fail")
		require.Contains(t, err.Error(), "dial unix all your base: connect: no such file or directory", "expected invalid node in config")
	})
}

func TestCallContractWithoutArgs(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newSimulatedEthereumConnectorHarness()
		initText := "are belong to us"
		methodToCall := "getValues"
		h.deploySimulatorStorageContract(ctx, initText)

		ethCallData, err := h.packInputArgumentsForSampleStorage(methodToCall, nil)
		require.NoError(t, err, "this means we couldn't pack the params for ethereum, something is broken with the harness")

		input := builders.EthereumCallContractInput().
			WithTimestamp(adapter.LastTimestampInFake.Add(-24 * time.Hour)).
			WithContractAddress(h.getAddress()).
			WithAbi(contract.SimpleStorageABI).
			WithFunctionName(methodToCall).
			WithPackedArguments(ethCallData).
			Build()

		output, err := h.connector.EthereumCallContract(ctx, input)
		require.NoError(t, err, "expecting call to succeed")
		require.True(t, len(output.EthereumAbiPackedOutput) > 0, "expecting output to have some data")

		t.Log(output.EthereumAbiPackedOutput)
		ret := new(struct { // this is the expected return type of that ethereum call for the SimpleStorage contract getValues
			IntValue    *big.Int
			StringValue string
		})

		parsedABI, _ := abi.JSON(strings.NewReader(contract.SimpleStorageABI))
		ethereum.ABIUnpackFunctionOutputArguments(parsedABI, ret, methodToCall, output.EthereumAbiPackedOutput)
		require.Equal(t, initText, ret.StringValue, "text part from eth")
	})
}

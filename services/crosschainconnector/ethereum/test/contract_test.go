package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
)

func TestContractCallBadNodeConfig(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newEthereumConnectorHarness().withInvalidEndpoint()

		input := builders.EthereumCallContractInput().Build() // don't care about specifics

		_, err := h.connector.EthereumCallContract(ctx, input)
		require.EqualError(t, err, "dial unix all your base: connect: no such file or directory", "expected invalid node in config")
	})
}

func TestCallContractWithoutArgs(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newEthereumConnectorHarness()
		initNum := int64(15)
		initText := "are belong to us"
		methodToCall := "getValues"
		h.deployStorageContract(ctx, initNum, initText)

		ethCallData, err := ethereumPackInputArguments(SimpleStorageABI, methodToCall, nil)
		require.NoError(t, err, "this means we couldn't pack the params for ethereum, something is broken with the harness")

		input := builders.EthereumCallContractInput().
			WithContractAddress(h.getAddress()).
			WithAbi(SimpleStorageABI).
			WithFunctionName(methodToCall).
			WithPackedArguments(ethCallData).
			Build()

		output, err := h.connector.EthereumCallContract(ctx, input)
		require.NoError(t, err, "expecting call to succeed")
		require.True(t, len(output.EthereumPackedOutput) > 0, "expecting output to have some data")
		t.Log(output.EthereumPackedOutput)
		ret := new(struct { // this is the expected return type of that ethereum call for the SimpleStorage contract getValues
			IntValue    *big.Int
			StringValue string
		})
		ethereumUnpackOutput(output.EthereumPackedOutput, methodToCall, ret)

		require.Equal(t, initNum, ret.IntValue.Int64(), "number part from eth")
		require.Equal(t, initText, ret.StringValue, "text")
	})
}

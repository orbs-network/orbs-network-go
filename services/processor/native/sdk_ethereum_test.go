package native

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"math/big"
	"strings"
	"testing"
)

// this example represents respones from "say" function with data "etherworld"
var examplePackedOutput = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 16, 104, 101, 108, 108, 111, 32, 101, 116, 104, 101, 114, 119, 111, 114, 108, 100, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

func TestEthereumSdk_CallMethod(t *testing.T) {
	s := createEthereumSdk()

	var out string
	sampleABI := "[{\"inputs\":[],\"name\":\"say\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"type\":\"function\"}]"
	s.SdkEthereumCallMethod(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SYSTEM, "ExampleAddress", sampleABI, "say", &out)
}

func createEthereumSdk() *service {
	return &service{sdkHandler: &contractSdkEthereumCallHandlerStub{}}
}

type contractSdkEthereumCallHandlerStub struct{}

func (c *contractSdkEthereumCallHandlerStub) HandleSdkCall(ctx context.Context, input *handlers.HandleSdkCallInput) (*handlers.HandleSdkCallOutput, error) {
	if input.PermissionScope != protocol.PERMISSION_SCOPE_SYSTEM {
		panic("permissions passed to SDK are incorrect")
	}
	switch input.MethodName {
	case "callMethod":
		return &handlers.HandleSdkCallOutput{
			OutputArguments: builders.MethodArguments(examplePackedOutput),
		}, nil
	default:
		return nil, errors.New("unknown method")
	}
}

func TestEthereumSdk_EthereumPackABINoArgs(t *testing.T) {
	sampleABI := "[{\"inputs\":[],\"name\":\"say\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"type\":\"function\"}]"
	parsedAbi := parseABIForPackingTests(t, sampleABI)
	methodNameInABI := "say"
	x, err := ethereumPackInputArguments(parsedAbi, methodNameInABI, nil)
	require.NoError(t, err, "failed to parse and pack the ABI")
	require.Equal(t, []byte{0x95, 0x4a, 0xb4, 0xb2}, x, "output byte array mismatch")
}

func parseABIForPackingTests(t *testing.T, jsonAbi string) abi.ABI {
	parsedABI, err := abi.JSON(strings.NewReader(jsonAbi))
	require.NoError(t, err, "problem parsing ABI")
	return parsedABI
}

func TestEthereumSdk_EthereumPackABIWithArgs(t *testing.T) {
	ABIStorage := "[{\"constant\":true,\"inputs\":[],\"name\":\"getValues\",\"outputs\":[{\"name\":\"intValue\",\"type\":\"uint256\"},{\"name\":\"stringValue\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getInt\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_multiple\",\"type\":\"uint256\"}],\"name\":\"getIntMultiple\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getString\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_intValue\",\"type\":\"uint256\"},{\"name\":\"_stringValue\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"}]"
	parsedABI := parseABIForPackingTests(t, ABIStorage)
	param1 := big.NewInt(2)
	args := []interface{}{param1}
	methodNameInABI := "getIntMultiple"
	x, err := ethereumPackInputArguments(parsedABI, methodNameInABI, args)
	require.NoError(t, err, "failed to parse and pack the ABI")
	expectedPackedBytes := []byte{0x82, 0xfa, 0x8a, 0xb2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2}
	require.Equal(t, expectedPackedBytes, x, "output byte array mismatch")
}

func TestEtherumSdk_EthereumUnpackData(t *testing.T) {
	ABIStorage := "[{\"constant\":true,\"inputs\":[],\"name\":\"getValues\",\"outputs\":[{\"name\":\"intValue\",\"type\":\"uint256\"},{\"name\":\"stringValue\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getInt\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_multiple\",\"type\":\"uint256\"}],\"name\":\"getIntMultiple\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getString\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_intValue\",\"type\":\"uint256\"},{\"name\":\"_stringValue\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"}]"
	parsedAbi := parseABIForPackingTests(t, ABIStorage)
	outputData := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 15, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 64, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 16, 97, 114, 101, 32, 98, 101, 108, 111, 110, 103, 32, 116, 111, 32, 117, 115, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	ret := new(struct { // this is the struct this data will fit into
		IntValue    *big.Int
		StringValue string
	})
	ethereumUnpackOutput(parsedAbi, ret, "getValues", outputData)
	require.EqualValues(t, 15, ret.IntValue.Int64(), "number part from eth")
	require.Equal(t, "are belong to us", ret.StringValue, "text part from eth")
}

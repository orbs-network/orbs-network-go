package native

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
)

var examplePackedOutput = []byte{0x01, 0x02, 0x03}

func TestEthereumSdk_CallMethod(t *testing.T) {
	s := createEthereumSdk()

	var out uint32
	sampleABI := "[{\"inputs\":[],\"name\":\"say\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"type\":\"function\"}]"
	err := s.CallMethod(EXAMPLE_CONTEXT, "ExampleAddress", sampleABI, "say", &out)
	require.NoError(t, err, "callMethod should succeed")
}

func createEthereumSdk() *ethereumSdk {
	return &ethereumSdk{
		handler:         &contractSdkEthereumCallHandlerStub{},
		permissionScope: protocol.PERMISSION_SCOPE_SYSTEM,
	}
}

type contractSdkEthereumCallHandlerStub struct {
}

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
	methodNameInABI := "say"
	x, err := ethereumPackInputArguments(sampleABI, methodNameInABI)
	require.NoError(t, err, "failed to parse and pack the ABI")
	require.Equal(t, []byte{0x95, 0x4a, 0xb4, 0xb2}, x, "output byte array mismatch")
}

func TestEthereumSdk_EthereumPackABIWithArgs(t *testing.T) {
	ABIStorage := "[{\"constant\":true,\"inputs\":[],\"name\":\"getValues\",\"outputs\":[{\"name\":\"intValue\",\"type\":\"uint256\"},{\"name\":\"stringValue\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getInt\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_multiple\",\"type\":\"uint256\"}],\"name\":\"getIntMultiple\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getString\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_intValue\",\"type\":\"uint256\"},{\"name\":\"_stringValue\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"}]"
	param1 := big.NewInt(2)
	methodNameInABI := "getIntMultiple"
	x, err := ethereumPackInputArguments(ABIStorage, methodNameInABI, param1)
	require.NoError(t, err, "failed to parse and pack the ABI")
	expectedPackedBytes := []byte{0x82, 0xfa, 0x8a, 0xb2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2}
	require.Equal(t, expectedPackedBytes, x, "output byte array mismatch")
}

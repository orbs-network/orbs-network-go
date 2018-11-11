package native

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

var examplePackedOutput = []byte{0x01, 0x02, 0x03}

func TestEthereumSdk_CallMethod(t *testing.T) {
	s := createEthereumSdk()

	var out uint32
	err := s.CallMethod(EXAMPLE_CONTEXT, "ExampleAddress", "ExampleAbi", "ExampleMethod", &out, "hello")
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

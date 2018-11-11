package native

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestServiceSdk_CallMethod(t *testing.T) {
	s := createServiceSdk()

	res, err := s.CallMethod(EXAMPLE_CONTEXT, "AnotherContract", "someMethod", uint64(17), "hello")
	require.NoError(t, err, "callMethod should succeed")
	require.Equal(t, []interface{}{uint64(17), "hello"}, res, "callMethod result should match expected")
}

func createServiceSdk() *serviceSdk {
	return &serviceSdk{
		handler:         &contractSdkServiceCallHandlerStub{},
		permissionScope: protocol.PERMISSION_SCOPE_SYSTEM,
	}
}

type contractSdkServiceCallHandlerStub struct {
}

func (c *contractSdkServiceCallHandlerStub) HandleSdkCall(ctx context.Context, input *handlers.HandleSdkCallInput) (*handlers.HandleSdkCallOutput, error) {
	if input.PermissionScope != protocol.PERMISSION_SCOPE_SYSTEM {
		panic("permissions passed to SDK are incorrect")
	}
	switch input.MethodName {
	case "callMethod":
		return &handlers.HandleSdkCallOutput{
			OutputArguments: []*protocol.MethodArgument{input.InputArguments[2]},
		}, nil
	default:
		return nil, errors.New("unknown method")
	}
}

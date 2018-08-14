package native

import (
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIsNative(t *testing.T) {
	s := createServiceSdk()

	err := s.IsNative(EXAMPLE_CONTEXT, "NativeContract")
	require.NoError(t, err, "isNative should succeed")

	err = s.IsNative(EXAMPLE_CONTEXT, "NonNativeContract")
	require.Error(t, err, "isNative should fail")
}

func TestCallMethod(t *testing.T) {
	s := createServiceSdk()

	err := s.CallMethod(EXAMPLE_CONTEXT, "AnotherContract", "someMethod")
	require.NoError(t, err, "callMethod should succeed")
}

func createServiceSdk() *serviceSdk {
	return &serviceSdk{
		handler: &contractSdkServiceCallHandlerStub{},
	}
}

type contractSdkServiceCallHandlerStub struct {
}

func (c *contractSdkServiceCallHandlerStub) HandleSdkCall(input *handlers.HandleSdkCallInput) (*handlers.HandleSdkCallOutput, error) {
	switch input.MethodName {
	case "isNative":
		if input.InputArguments[0].StringValue() == "NativeContract" {
			return nil, nil
		} else {
			return nil, errors.New("not native contract")
		}
	case "callMethod":
		return nil, nil
	default:
		return nil, errors.New("unknown method")
	}
}

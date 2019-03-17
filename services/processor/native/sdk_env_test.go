package native

import (
	"context"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSdkEnv_GetBlockHeight(t *testing.T) {
	s := createEnvSdk()

	height := s.SdkEnvGetBlockHeight(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SERVICE)
	require.EqualValues(t, height, uint64(11), "block height should be returned")
}

func TestSdkEnv_GetBlockTimestamp(t *testing.T) {
	s := createEnvSdk()

	height := s.SdkEnvGetBlockTimestamp(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SERVICE)
	require.EqualValues(t, height, uint64(12), "block timestamp should be returned")
}

func TestSdkEnv_GetVirtualChainId(t *testing.T) {
	s := &service{config: config.ForNativeProcessorTests(42)}
	vcid := s.SdkEnvGetVirtualChainId(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SERVICE)
	require.EqualValues(t, vcid, 42, "virtual chain id should be returned")

}

func createEnvSdk() *service {
	return &service{sdkHandler: &contractSdkEnvCallHandlerStub{}}
}

type contractSdkEnvCallHandlerStub struct{}

func (c *contractSdkEnvCallHandlerStub) HandleSdkCall(ctx context.Context, input *handlers.HandleSdkCallInput) (*handlers.HandleSdkCallOutput, error) {
	if input.PermissionScope != protocol.PERMISSION_SCOPE_SERVICE {
		panic("permissions passed to SDK are incorrect")
	}
	switch input.MethodName {
	case "getBlockHeight":
		return &handlers.HandleSdkCallOutput{
			OutputArguments: builders.Arguments(uint64(11)),
		}, nil
	case "getBlockTimestamp":
		return &handlers.HandleSdkCallOutput{
			OutputArguments: builders.Arguments(uint64(12)),
		}, nil
	default:
		return nil, errors.New("unknown method")
	}
}

package native

import (
	"context"
	"encoding/hex"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

var exampleAddress1, _ = hex.DecodeString("1acb19a469206161ed7e5ed9feb996a6e24be441") // ripmd160
var exampleAddress2, _ = hex.DecodeString("223344a469206161ed7e5ed9feb996a6e24be441") // ripmd160

func TestSdkAddress_GetSignerAddress(t *testing.T) {
	s := createAddressSdk()

	address := s.SdkAddressGetSignerAddress(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SERVICE)
	require.EqualValues(t, exampleAddress1, address, "example1 should be returned")
}

func TestSdkAddress_GetCallerAddress(t *testing.T) {
	s := createAddressSdk()

	address := s.SdkAddressGetCallerAddress(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SERVICE)
	require.EqualValues(t, exampleAddress2, address, "example2 should be returned")
}

func createAddressSdk() *service {
	return &service{sdkHandler: &contractSdkAddressCallHandlerStub{}}
}

type contractSdkAddressCallHandlerStub struct{}

func (c *contractSdkAddressCallHandlerStub) HandleSdkCall(ctx context.Context, input *handlers.HandleSdkCallInput) (*handlers.HandleSdkCallOutput, error) {
	if input.PermissionScope != protocol.PERMISSION_SCOPE_SERVICE {
		panic("permissions passed to SDK are incorrect")
	}
	switch input.MethodName {
	case "getSignerAddress":
		return &handlers.HandleSdkCallOutput{
			OutputArguments: builders.MethodArguments(exampleAddress1),
		}, nil
	case "getCallerAddress":
		return &handlers.HandleSdkCallOutput{
			OutputArguments: builders.MethodArguments(exampleAddress2),
		}, nil
	default:
		return nil, errors.New("unknown method")
	}
}

package native

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

// TODO: change both examples to valid ripmd160 hashes once we fix the calculation
var exampleAddress1, _ = hex.DecodeString("cf80cd8aed482d5d1527d7dc72fceff84e6326592848447d2dc0b0e87dfc9a909c1185a5c5e9fc54612808977ee8f548b2258d31") // ripmd160
var exampleAddress2, _ = hex.DecodeString("2233448aed482d5d1527d7dc72fceff84e6326592848447d2dc0b0e87dfc9a909c1185a5c5e9fc54612808977ee8f548b2258d31") // ripmd160

func TestValidateAddress(t *testing.T) {
	s := createAddressSdk()

	err := s.ValidateAddress(EXAMPLE_CONTEXT, []byte{})
	require.Error(t, err, "address should not be valid since empty")

	err = s.ValidateAddress(EXAMPLE_CONTEXT, []byte{0x01, 0x02, 0x03})
	require.Error(t, err, "address should not be valid since length is invalid")

	err = s.ValidateAddress(EXAMPLE_CONTEXT, exampleAddress1)
	require.NoError(t, err, "address should be valid")
}

func TestGetSignerAddress(t *testing.T) {
	s := createAddressSdk()

	address, err := s.GetSignerAddress(EXAMPLE_CONTEXT)
	require.NoError(t, err, "call should be successful")
	require.EqualValues(t, exampleAddress1, address, "example1 should be returned")
}

func TestGetCallerAddress(t *testing.T) {
	s := createAddressSdk()

	address, err := s.GetCallerAddress(EXAMPLE_CONTEXT)
	require.NoError(t, err, "call should be successful")
	require.EqualValues(t, exampleAddress2, address, "example2 should be returned")
}

func createAddressSdk() *addressSdk {
	return &addressSdk{
		handler:         &contractSdkAddressCallHandlerStub{},
		permissionScope: protocol.PERMISSION_SCOPE_SERVICE,
	}
}

type contractSdkAddressCallHandlerStub struct {
}

func (c *contractSdkAddressCallHandlerStub) HandleSdkCall(input *handlers.HandleSdkCallInput) (*handlers.HandleSdkCallOutput, error) {
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

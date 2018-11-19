package native

import (
	"context"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

const EXAMPLE_CONTEXT = 0

func exampleKey() string {
	return "example-key"
}

func exampleKeyAddress() []byte {
	return hash.CalcRipmd160Sha256([]byte(exampleKey()))
}

func TestWriteReadBytesByAddress(t *testing.T) {
	s := createStateSdk()
	s.SdkStateWriteBytesByAddress(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SERVICE, exampleKeyAddress(), []byte{0x01, 0x02, 0x03})

	bytes := s.SdkStateReadBytesByAddress(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SERVICE, exampleKeyAddress())
	require.Equal(t, []byte{0x01, 0x02, 0x03}, bytes, "read should return what was written")
}

func createStateSdk() *service {
	return &service{sdkHandler: &contractSdkStateCallHandlerStub{}}
}

type contractSdkStateCallHandlerStub struct {
	store map[string]*protocol.MethodArgument
}

func (c *contractSdkStateCallHandlerStub) HandleSdkCall(ctx context.Context, input *handlers.HandleSdkCallInput) (*handlers.HandleSdkCallOutput, error) {
	if input.PermissionScope != protocol.PERMISSION_SCOPE_SERVICE {
		panic("permissions passed to SDK are incorrect")
	}
	switch input.MethodName {
	case "read":
		return &handlers.HandleSdkCallOutput{
			OutputArguments: []*protocol.MethodArgument{c.store[string(input.InputArguments[0].BytesValue())]},
		}, nil
	case "write":
		c.store[string(input.InputArguments[0].BytesValue())] = input.InputArguments[1]
		return nil, nil
	default:
		return nil, errors.New("unknown method")
	}
}

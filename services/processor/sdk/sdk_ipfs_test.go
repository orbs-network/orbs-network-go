package sdk

import (
	"context"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSdkIPFS_Read(t *testing.T) {
	s := createIPFSSdk()

	contents := s.SdkIPFSRead(EXAMPLE_CONTEXT, sdkContext.PERMISSION_SCOPE_SERVICE, []byte("an album"))
	require.EqualValues(t, contents, []byte("Diamond Dogs"), "file contents should be returned")
}

func createIPFSSdk() *service {
	return &service{sdkHandler: &contractSdkIPFSCallHandlerStub{}}
}

type contractSdkIPFSCallHandlerStub struct{}

func (c *contractSdkIPFSCallHandlerStub) HandleSdkCall(ctx context.Context, input *handlers.HandleSdkCallInput) (*handlers.HandleSdkCallOutput, error) {
	if input.PermissionScope != protocol.PERMISSION_SCOPE_SERVICE {
		panic("permissions passed to SDK are incorrect")
	}
	var readValue interface{}
	switch input.MethodName {
	case "read":
		readValue = []byte("Diamond Dogs")
	default:
		return nil, errors.New("unknown method")
	}
	outputArgs, err := protocol.ArgumentsFromNatives([]interface{}{readValue})
	if err != nil {
		return nil, errors.Wrapf(err, "unknown input arg")
	}
	return &handlers.HandleSdkCallOutput{OutputArguments: outputArgs}, nil
}

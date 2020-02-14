package sdk

import (
	"context"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

const SDK_OPERATION_NAME_IPFS = "Sdk.IPFS"

func (s *service) SdkIPFSRead(executionContextId sdkContext.ContextId, permissionScope sdkContext.PermissionScope, hash []byte) []byte {
	args, err := protocol.ArgumentsFromNatives(builders.VarsToSlice(hash))
	if err != nil {
		panic(err)
	}
	output, err := s.sdkHandler.HandleSdkCall(context.TODO(), &handlers.HandleSdkCallInput{
		ContextId:       primitives.ExecutionContextId(executionContextId),
		OperationName:   SDK_OPERATION_NAME_IPFS,
		MethodName:      "read",
		InputArguments:  args,
		PermissionScope: protocol.ExecutionPermissionScope(permissionScope),
	})
	if err != nil {
		panic(err.Error())
	}

	if len(output.OutputArguments) != 1 || !output.OutputArguments[0].IsTypeBytesValue() {
		panic("read Sdk.IPFS returned corrupt output value")
	}
	return output.OutputArguments[0].BytesValue()
}

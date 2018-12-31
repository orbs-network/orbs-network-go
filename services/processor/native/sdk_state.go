package native

import (
	"context"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

const SDK_OPERATION_NAME_STATE = "Sdk.State"

func (s *service) SdkStateReadBytesByAddress(executionContextId sdkContext.ContextId, permissionScope sdkContext.PermissionScope, address []byte) []byte {
	output, err := s.sdkHandler.HandleSdkCall(context.TODO(), &handlers.HandleSdkCallInput{
		ContextId:     primitives.ExecutionContextId(executionContextId),
		OperationName: SDK_OPERATION_NAME_STATE,
		MethodName:    "read",
		InputArguments: []*protocol.Argument{
			(&protocol.ArgumentBuilder{
				// key
				Type:       protocol.ARGUMENT_TYPE_BYTES_VALUE,
				BytesValue: address,
			}).Build(),
		},
		PermissionScope: protocol.ExecutionPermissionScope(permissionScope),
	})
	if err != nil {
		panic(err.Error())
	}
	if len(output.OutputArguments) != 1 || !output.OutputArguments[0].IsTypeBytesValue() {
		panic("read Sdk.State returned corrupt output value")
	}
	return output.OutputArguments[0].BytesValue()
}

func (s *service) SdkStateWriteBytesByAddress(executionContextId sdkContext.ContextId, permissionScope sdkContext.PermissionScope, address []byte, value []byte) {
	_, err := s.sdkHandler.HandleSdkCall(context.TODO(), &handlers.HandleSdkCallInput{
		ContextId:     primitives.ExecutionContextId(executionContextId),
		OperationName: SDK_OPERATION_NAME_STATE,
		MethodName:    "write",
		InputArguments: []*protocol.Argument{
			(&protocol.ArgumentBuilder{
				// key
				Type:       protocol.ARGUMENT_TYPE_BYTES_VALUE,
				BytesValue: address,
			}).Build(),
			(&protocol.ArgumentBuilder{
				// value
				Type:       protocol.ARGUMENT_TYPE_BYTES_VALUE,
				BytesValue: value,
			}).Build(),
		},
		PermissionScope: protocol.ExecutionPermissionScope(permissionScope),
	})
	if err != nil {
		panic(err.Error())
	}
}

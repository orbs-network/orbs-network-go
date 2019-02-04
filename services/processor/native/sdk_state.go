package native

import (
	"context"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

const SDK_OPERATION_NAME_STATE = "Sdk.State"

func (s *service) SdkStateReadBytes(executionContextId sdkContext.ContextId, permissionScope sdkContext.PermissionScope, key []byte) []byte {
	output, err := s.sdkHandler.HandleSdkCall(context.TODO(), &handlers.HandleSdkCallInput{
		ContextId:     primitives.ExecutionContextId(executionContextId),
		OperationName: SDK_OPERATION_NAME_STATE,
		MethodName:    "read",
		InputArguments: []*protocol.Argument{
			(&protocol.ArgumentBuilder{
				// key
				Type:       protocol.ARGUMENT_TYPE_BYTES_VALUE,
				BytesValue: key,
			}).Build(),
		},
		PermissionScope: protocol.ExecutionPermissionScope(permissionScope),
	})
	if err != nil {
		s.logger.Panic("failed handling SDK call", log.Error(err))
	}
	if len(output.OutputArguments) != 1 || !output.OutputArguments[0].IsTypeBytesValue() {
		s.logger.Panic("read Sdk.State returned corrupt output value", log.StringableSlice("output-arguments", output.OutputArguments))
	}
	return output.OutputArguments[0].BytesValue()
}

func (s *service) SdkStateWriteBytes(executionContextId sdkContext.ContextId, permissionScope sdkContext.PermissionScope, key []byte, value []byte) {
	_, err := s.sdkHandler.HandleSdkCall(context.TODO(), &handlers.HandleSdkCallInput{
		ContextId:     primitives.ExecutionContextId(executionContextId),
		OperationName: SDK_OPERATION_NAME_STATE,
		MethodName:    "write",
		InputArguments: []*protocol.Argument{
			(&protocol.ArgumentBuilder{
				// key
				Type:       protocol.ARGUMENT_TYPE_BYTES_VALUE,
				BytesValue: key,
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
		s.logger.Panic("failed handling SDK call", log.Error(err))
	}
}

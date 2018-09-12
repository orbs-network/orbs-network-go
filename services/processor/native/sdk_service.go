package native

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

type serviceSdk struct {
	handler         handlers.ContractSdkCallHandler
	permissionScope protocol.ExecutionPermissionScope
}

const SDK_OPERATION_NAME_SERVICE = "Sdk.Service"

func (s *serviceSdk) CallMethod(executionContextId types.Context, serviceName string, methodName string) error {
	_, err := s.handler.HandleSdkCall(&handlers.HandleSdkCallInput{
		ContextId:     primitives.ExecutionContextId(executionContextId),
		OperationName: SDK_OPERATION_NAME_SERVICE,
		MethodName:    "callMethod",
		InputArguments: []*protocol.MethodArgument{
			(&protocol.MethodArgumentBuilder{
				Name:        "serviceName",
				Type:        protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE,
				StringValue: serviceName,
			}).Build(),
			(&protocol.MethodArgumentBuilder{
				Name:        "methodName",
				Type:        protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE,
				StringValue: methodName,
			}).Build(),
		},
		PermissionScope: s.permissionScope,
	})
	return err
}

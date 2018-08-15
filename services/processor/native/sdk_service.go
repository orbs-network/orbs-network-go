package native

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

type serviceSdk struct {
	handler handlers.ContractSdkCallHandler
}

const SDK_SERVICE_CONTRACT_NAME = "Sdk.Service"

func (s *serviceSdk) IsNative(ctx types.Context, serviceName string) error {
	_, err := s.handler.HandleSdkCall(&handlers.HandleSdkCallInput{
		ContextId:    primitives.ExecutionContextId(ctx),
		ContractName: SDK_SERVICE_CONTRACT_NAME,
		MethodName:   "isNative",
		InputArguments: []*protocol.MethodArgument{
			(&protocol.MethodArgumentBuilder{
				Name:        "serviceName",
				Type:        protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE,
				StringValue: serviceName,
			}).Build(),
		},
	})
	return err
}

func (s *serviceSdk) CallMethod(ctx types.Context, serviceName string, methodName string) error {
	_, err := s.handler.HandleSdkCall(&handlers.HandleSdkCallInput{
		ContextId:    primitives.ExecutionContextId(ctx),
		ContractName: SDK_SERVICE_CONTRACT_NAME,
		MethodName:   "callMethod",
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
	})
	return err
}

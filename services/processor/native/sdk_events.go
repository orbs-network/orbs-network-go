package native

import (
	"context"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
)

const SDK_OPERATION_NAME_EVENTS = "Sdk.Events"

func (s *service) SdkEventsEmitEvent(executionContextId sdkContext.ContextId, permissionScope sdkContext.PermissionScope, eventFunctionSignature interface{}, args ...interface{}) {
	eventName, err := types.GetContractMethodNameFromFunction(eventFunctionSignature)
	if err != nil {
		panic(err.Error())
	}

	argsArgumentArray := argsToMethodArgumentArray(args...)
	err = s.validateEventInputArgs(eventFunctionSignature, argsArgumentArray)
	if err != nil {
		panic(errors.Wrap(err, "incorrect types given to event emit"))
	}

	_, err = s.sdkHandler.HandleSdkCall(context.TODO(), &handlers.HandleSdkCallInput{
		ContextId:     primitives.ExecutionContextId(executionContextId),
		OperationName: SDK_OPERATION_NAME_EVENTS,
		MethodName:    "emitEvent",
		InputArguments: []*protocol.MethodArgument{
			(&protocol.MethodArgumentBuilder{
				Name:        "eventName",
				Type:        protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE,
				StringValue: eventName,
			}).Build(),
			(&protocol.MethodArgumentBuilder{
				Name:       "inputArgs",
				Type:       protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE,
				BytesValue: argsArgumentArray.Raw(),
			}).Build(),
		},
		PermissionScope: protocol.ExecutionPermissionScope(permissionScope),
	})
	if err != nil {
		panic(err.Error())
	}
}

func (s *service) validateEventInputArgs(eventFunctionSignature interface{}, argsArgumentArray *protocol.MethodArgumentArray) error {
	_, err := s.prepareMethodInputArgsForCall(eventFunctionSignature, argsArgumentArray)
	return err
}

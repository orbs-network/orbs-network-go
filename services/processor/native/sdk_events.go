package native

import (
	"context"
	"fmt"
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

	functionNameForErrors := fmt.Sprintf("EVENTS.%s", eventName)
	argsArgumentArray := argsToArgumentArray(args...)
	err = s.validateEventInputArgs(eventFunctionSignature, argsArgumentArray, functionNameForErrors)
	if err != nil {
		panic(errors.Wrap(err, "incorrect types given to event emit").Error())
	}

	_, err = s.sdkHandler.HandleSdkCall(context.TODO(), &handlers.HandleSdkCallInput{
		ContextId:     primitives.ExecutionContextId(executionContextId),
		OperationName: SDK_OPERATION_NAME_EVENTS,
		MethodName:    "emitEvent",
		InputArguments: []*protocol.Argument{
			(&protocol.ArgumentBuilder{
				// eventName
				Type:        protocol.ARGUMENT_TYPE_STRING_VALUE,
				StringValue: eventName,
			}).Build(),
			(&protocol.ArgumentBuilder{
				// inputArgs
				Type:       protocol.ARGUMENT_TYPE_BYTES_VALUE,
				BytesValue: argsArgumentArray.Raw(),
			}).Build(),
		},
		PermissionScope: protocol.ExecutionPermissionScope(permissionScope),
	})
	if err != nil {
		panic(err.Error())
	}
}

func (s *service) validateEventInputArgs(eventFunctionSignature interface{}, argsArgumentArray *protocol.ArgumentArray, functionNameForErrors string) error {
	_, err := s.prepareMethodInputArgsForCall(eventFunctionSignature, argsArgumentArray, functionNameForErrors)
	return err
}

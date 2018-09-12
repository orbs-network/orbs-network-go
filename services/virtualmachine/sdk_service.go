package virtualmachine

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

func (s *service) handleSdkServiceCall(context *executionContext, methodName primitives.MethodName, args []*protocol.MethodArgument, permissionScope protocol.ExecutionPermissionScope) ([]*protocol.MethodArgument, error) {
	switch methodName {

	case "callMethod":
		outputArgumentArrayRaw, err := s.handleSdkServiceCallMethod(context, args, permissionScope)
		return []*protocol.MethodArgument{(&protocol.MethodArgumentBuilder{
			Name:       "outputArgs",
			Type:       protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE,
			BytesValue: outputArgumentArrayRaw,
		}).Build()}, err

	default:
		return nil, errors.Errorf("unknown SDK service call method: %s", methodName)
	}
}

// inputArg0: serviceName (string)
// inputArg1: methodName (string)
// inputArg2: inputArgumentArray ([]byte of raw MethodArgumentArray)
// outputArg0: outputArgumentArray ([]byte of raw MethodArgumentArray)
func (s *service) handleSdkServiceCallMethod(context *executionContext, args []*protocol.MethodArgument, permissionScope protocol.ExecutionPermissionScope) ([]byte, error) {
	if len(args) != 3 || !args[0].IsTypeStringValue() || !args[1].IsTypeStringValue() || !args[2].IsTypeBytesValue() {
		return nil, errors.Errorf("invalid SDK service callMethod args: %v", args)
	}
	serviceName := args[0].StringValue()
	methodName := args[1].StringValue()
	inputArgumentArray := protocol.MethodArgumentArrayReader(args[2].BytesValue())

	// get deployment info
	processor, err := s.getServiceDeployment(context, primitives.ContractName(serviceName))
	if err != nil {
		s.reporting.Info("get deployment info for contract failed during Sdk.Service.CallMethod", log.Error(err), log.String("contract", serviceName))
		return nil, err
	}

	// modify execution context
	callingService := context.serviceStackTop()
	context.serviceStackPush(primitives.ContractName(serviceName))
	defer context.serviceStackPop()

	// execute the call
	output, err := processor.ProcessCall(&services.ProcessCallInput{
		ContextId:              context.contextId,
		ContractName:           primitives.ContractName(serviceName),
		MethodName:             primitives.MethodName(methodName),
		InputArgumentArray:     inputArgumentArray,
		AccessScope:            context.accessScope,
		CallingPermissionScope: permissionScope,
		CallingService:         callingService,
		TransactionSigner:      nil,
	})
	if err != nil {
		s.reporting.Info("Sdk.Service.CallMethod failed", log.Error(err), log.Stringable("caller", callingService), log.Stringable("callee", primitives.ContractName(serviceName)))
		return nil, err
	}

	return output.OutputArgumentArray.Raw(), nil
}

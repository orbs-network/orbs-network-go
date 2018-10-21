package virtualmachine

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

func (s *service) handleSdkServiceCall(ctx context.Context, executionContext *executionContext, methodName primitives.MethodName, args []*protocol.MethodArgument, permissionScope protocol.ExecutionPermissionScope) ([]*protocol.MethodArgument, error) {
	switch methodName {

	case "callMethod":
		outputArgumentArrayRaw, err := s.handleSdkServiceCallMethod(ctx, executionContext, args, permissionScope)
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
func (s *service) handleSdkServiceCallMethod(ctx context.Context, executionContext *executionContext, args []*protocol.MethodArgument, permissionScope protocol.ExecutionPermissionScope) ([]byte, error) {
	if len(args) != 3 || !args[0].IsTypeStringValue() || !args[1].IsTypeStringValue() || !args[2].IsTypeBytesValue() {
		return nil, errors.Errorf("invalid SDK service callMethod args: %v", args)
	}
	serviceName := args[0].StringValue()
	methodName := args[1].StringValue()
	inputArgumentArray := protocol.MethodArgumentArrayReader(args[2].BytesValue())

	// get deployment info
	processor, err := s.getServiceDeployment(ctx, executionContext, primitives.ContractName(serviceName))
	if err != nil {
		s.logger.Info("get deployment info for contract failed during Sdk.Service.CallMethod", log.Error(err), log.String("contract", serviceName))
		return nil, err
	}

	// modify execution context
	callingService := executionContext.serviceStackTop()
	executionContext.serviceStackPush(primitives.ContractName(serviceName))
	defer executionContext.serviceStackPop()

	// execute the call
	output, err := processor.ProcessCall(ctx, &services.ProcessCallInput{
		ContextId:              executionContext.contextId,
		ContractName:           primitives.ContractName(serviceName),
		MethodName:             primitives.MethodName(methodName),
		InputArgumentArray:     inputArgumentArray,
		AccessScope:            executionContext.accessScope,
		CallingPermissionScope: permissionScope,
		CallingService:         callingService,
	})
	if err != nil {
		s.logger.Info("Sdk.Service.CallMethod failed", log.Error(err), log.Stringable("caller", callingService), log.Stringable("callee", primitives.ContractName(serviceName)))
		return nil, err
	}

	return output.OutputArgumentArray.Raw(), nil
}

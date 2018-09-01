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
		err := s.handleSdkServiceCallMethod(context, args, permissionScope)
		return []*protocol.MethodArgument{}, err

	default:
		return nil, errors.Errorf("unknown SDK service call method: %s", methodName)
	}
}

func (s *service) handleSdkServiceCallMethod(context *executionContext, args []*protocol.MethodArgument, permissionScope protocol.ExecutionPermissionScope) error {
	if len(args) != 2 || !args[0].IsTypeStringValue() || !args[1].IsTypeStringValue() {
		return errors.Errorf("invalid SDK service callMethod args: %v", args)
	}
	serviceName := args[0].StringValue()
	methodName := args[1].StringValue()

	// modify execution context
	callingService := context.serviceStackTop()
	context.serviceStackPush(primitives.ContractName(serviceName))
	defer context.serviceStackPop()

	// execute the call
	_, err := s.processors[protocol.PROCESSOR_TYPE_NATIVE].ProcessCall(&services.ProcessCallInput{
		ContextId:              context.contextId,
		ContractName:           primitives.ContractName(serviceName),
		MethodName:             primitives.MethodName(methodName),
		InputArgumentArray:     (&protocol.MethodArgumentArrayBuilder{}).Build(), // TODO: support args
		AccessScope:            context.accessScope,
		CallingPermissionScope: permissionScope,
		CallingService:         callingService,
		TransactionSigner:      nil,
	})
	if err != nil {
		s.reporting.Info("Sdk.Service.CallMethod failed", log.Error(err), log.Stringable("caller", callingService), log.Stringable("callee", primitives.ContractName(serviceName)))
	}

	return err // TODO: support result
}

package virtualmachine

import (
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

func (s *service) handleSdkServiceCall(context *executionContext, methodName primitives.MethodName, args []*protocol.MethodArgument) ([]*protocol.MethodArgument, error) {
	switch methodName {

	case "isNative":
		err := s.handleSdkServiceIsNative(context, args)
		return []*protocol.MethodArgument{}, err

	case "callMethod":
		err := s.handleSdkServiceCallMethod(context, args)
		return []*protocol.MethodArgument{}, err

	default:
		return nil, errors.Errorf("unknown SDK service call method: %s", methodName)
	}
}

func (s *service) handleSdkServiceIsNative(context *executionContext, args []*protocol.MethodArgument) error {
	if len(args) != 1 || !args[0].IsTypeStringValue() {
		return errors.Errorf("invalid SDK service isNative args: %v", args)
	}
	serviceName := args[0].StringValue()

	_, err := s.processors[protocol.PROCESSOR_TYPE_NATIVE].GetContractInfo(&services.GetContractInfoInput{
		ContractName: primitives.ContractName(serviceName),
	})

	return err
}

func (s *service) handleSdkServiceCallMethod(context *executionContext, args []*protocol.MethodArgument) error {
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
		ContextId:         context.contextId,
		ContractName:      primitives.ContractName(serviceName),
		MethodName:        primitives.MethodName(methodName),
		InputArguments:    []*protocol.MethodArgument{}, // TODO: support args
		AccessScope:       context.accessScope,
		PermissionScope:   protocol.PERMISSION_SCOPE_SERVICE, // TODO: kill this arg
		CallingService:    callingService,
		TransactionSigner: nil,
	})
	if err != nil {
		s.reporting.Info("Sdk.Service.CallMethod failed", instrumentation.Error(err), instrumentation.Stringable("caller", callingService), instrumentation.Stringable("callee", primitives.ContractName(serviceName)))
	}

	return err
}

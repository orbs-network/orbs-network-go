package virtualmachine

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

func (s *service) getServiceDeployment(executionContext *executionContext, serviceName primitives.ContractName) (services.Processor, error) {
	// call the system contract to identify the processor
	processorType, err := s.callIsServiceDeployedContract(executionContext, serviceName)
	if err != nil {
		return nil, err
	}

	// return according to processor
	switch processorType {
	case protocol.PROCESSOR_TYPE_NATIVE:
		return s.processors[protocol.PROCESSOR_TYPE_NATIVE], nil
	default:
		return nil, errors.Errorf("isServiceDeployed contract returned unknown processor type: %s", processorType)
	}
}

func (s *service) callIsServiceDeployedContract(executionContext *executionContext, serviceName primitives.ContractName) (protocol.ProcessorType, error) {
	systemContractName := deployments.CONTRACT.Name
	systemMethodName := deployments.METHOD_IS_SERVICE_DEPLOYED_READ_ONLY.Name
	if executionContext.accessScope == protocol.ACCESS_SCOPE_READ_WRITE {
		systemMethodName = deployments.METHOD_IS_SERVICE_DEPLOYED.Name
	}

	// modify execution context
	executionContext.serviceStackPush(systemContractName)
	defer executionContext.serviceStackPop()

	// execute the call
	output, err := s.processors[protocol.PROCESSOR_TYPE_NATIVE].ProcessCall(&services.ProcessCallInput{
		ContextId:    executionContext.contextId,
		ContractName: systemContractName,
		MethodName:   systemMethodName,
		InputArguments: []*protocol.MethodArgument{(&protocol.MethodArgumentBuilder{
			Name:        "serviceName",
			Type:        protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE,
			StringValue: string(serviceName),
		}).Build()},
		AccessScope:            executionContext.accessScope,
		CallingPermissionScope: protocol.PERMISSION_SCOPE_SERVICE,
		CallingService:         systemContractName,
		TransactionSigner:      nil,
	})
	if err != nil {
		return 0, err
	}
	if len(output.OutputArguments) != 1 || !output.OutputArguments[0].IsTypeUint32Value() {
		return 0, errors.Errorf("isServiceDeployed contract returned corrupt output value")
	}
	return protocol.ProcessorType(output.OutputArguments[0].Uint32Value()), nil
}

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
	processorType, err := s.callGetInfoOfDeploymentSystemContract(executionContext, serviceName)

	// on failure (contract not deployed), attempt to auto deploy native contract
	if err != nil {
		processorType, err = s.attemptToAutoDeployNativeContract(executionContext, serviceName)
		if err != nil {
			return nil, err
		}
	}

	// return according to processor
	switch processorType {
	case protocol.PROCESSOR_TYPE_NATIVE:
		return s.processors[protocol.PROCESSOR_TYPE_NATIVE], nil
	default:
		return nil, errors.Errorf("_Deployments.getInfo contract returned unknown processor type: %s", processorType)
	}
}

func (s *service) attemptToAutoDeployNativeContract(executionContext *executionContext, serviceName primitives.ContractName) (protocol.ProcessorType, error) {
	// make sure we have a write context (needed for deployment)
	if executionContext.accessScope != protocol.ACCESS_SCOPE_READ_WRITE {
		return 0, errors.Errorf("context accessScope is %s instead of read-write needed for auto deployment", executionContext.accessScope)
	}

	// make sure this is a native contract
	_, err := s.processors[protocol.PROCESSOR_TYPE_NATIVE].GetContractInfo(&services.GetContractInfoInput{
		ContractName: serviceName,
	})
	if err != nil {
		return 0, errors.Wrap(err, "attempting to auto deploy native contract")
	}

	// auto deploy native contract
	err = s.callDeployServiceOfDeploymentSystemContract(executionContext, serviceName)
	if err != nil {
		return 0, err
	}

	// auto deploy native contract was successful
	return protocol.PROCESSOR_TYPE_NATIVE, nil
}

func (s *service) callGetInfoOfDeploymentSystemContract(executionContext *executionContext, serviceName primitives.ContractName) (protocol.ProcessorType, error) {
	systemContractName := primitives.ContractName(deployments_systemcontract.CONTRACT.Name)
	systemMethodName := primitives.MethodName(deployments_systemcontract.METHOD_GET_INFO.Name)

	// modify execution context
	executionContext.serviceStackPush(systemContractName)
	defer executionContext.serviceStackPop()

	// execute the call
	inputArgs := (&protocol.MethodArgumentArrayBuilder{
		Arguments: []*protocol.MethodArgumentBuilder{
			{
				Name:        "serviceName",
				Type:        protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE,
				StringValue: string(serviceName),
			},
		},
	}).Build()
	output, err := s.processors[protocol.PROCESSOR_TYPE_NATIVE].ProcessCall(&services.ProcessCallInput{
		ContextId:              executionContext.contextId,
		ContractName:           systemContractName,
		MethodName:             systemMethodName,
		InputArgumentArray:     inputArgs,
		AccessScope:            executionContext.accessScope,
		CallingPermissionScope: protocol.PERMISSION_SCOPE_SERVICE,
		CallingService:         systemContractName,
		TransactionSigner:      nil,
	})
	if err != nil {
		return 0, err
	}
	outputArgsIterator := output.OutputArgumentArray.ArgumentsIterator()
	if !outputArgsIterator.HasNext() {
		return 0, errors.Errorf("_Deployments.getInfo contract returned corrupt output value")
	}
	outputArg0 := outputArgsIterator.NextArguments()
	if !outputArg0.IsTypeUint32Value() {
		return 0, errors.Errorf("_Deployments.getInfo contract returned corrupt output value")
	}
	return protocol.ProcessorType(outputArg0.Uint32Value()), nil
}

func (s *service) callDeployServiceOfDeploymentSystemContract(executionContext *executionContext, serviceName primitives.ContractName) error {
	systemContractName := primitives.ContractName(deployments_systemcontract.CONTRACT.Name)
	systemMethodName := primitives.MethodName(deployments_systemcontract.METHOD_DEPLOY_SERVICE.Name)

	// modify execution context
	executionContext.serviceStackPush(systemContractName)
	defer executionContext.serviceStackPop()

	// execute the call
	inputArgs := (&protocol.MethodArgumentArrayBuilder{
		Arguments: []*protocol.MethodArgumentBuilder{
			{
				Name:        "serviceName",
				Type:        protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE,
				StringValue: string(serviceName),
			},
			{
				Name:        "processorType",
				Type:        protocol.METHOD_ARGUMENT_TYPE_UINT_32_VALUE,
				Uint32Value: uint32(protocol.PROCESSOR_TYPE_NATIVE),
			},
			{
				Name:       "code",
				Type:       protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE,
				BytesValue: []byte{},
			},
		},
	}).Build()
	_, err := s.processors[protocol.PROCESSOR_TYPE_NATIVE].ProcessCall(&services.ProcessCallInput{
		ContextId:              executionContext.contextId,
		ContractName:           systemContractName,
		MethodName:             systemMethodName,
		InputArgumentArray:     inputArgs,
		AccessScope:            executionContext.accessScope,
		CallingPermissionScope: protocol.PERMISSION_SCOPE_SERVICE,
		CallingService:         systemContractName,
		TransactionSigner:      nil,
	})
	if err != nil {
		return err
	}
	return nil
}

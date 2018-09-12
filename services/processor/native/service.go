package native

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"sync"
)

type service struct {
	reporting log.BasicLogger

	mutex                        *sync.RWMutex
	contractSdkHandlerUnderMutex handlers.ContractSdkCallHandler
	contractRepositoryUnderMutex map[primitives.ContractName]types.Contract
}

func NewNativeProcessor(
	reporting log.BasicLogger,
) services.Processor {
	return &service{
		reporting: reporting.For(log.Service("processor-native")),
		mutex:     &sync.RWMutex{},
	}
}

// runs once on system initialization (called by the virtual machine constructor)
func (s *service) RegisterContractSdkCallHandler(handler handlers.ContractSdkCallHandler) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.contractSdkHandlerUnderMutex = handler
	s.contractRepositoryUnderMutex = s.initializePreBuiltRepositoryContractInstances(handler)
}

func (s *service) ProcessCall(input *services.ProcessCallInput) (*services.ProcessCallOutput, error) {
	// retrieve code
	executionContextId := types.Context(input.ContextId)
	contractInfo, methodInfo, err := s.retrieveContractAndMethodInfoFromRepository(executionContextId, input.ContractName, input.MethodName)
	if err != nil {
		return &services.ProcessCallOutput{
			OutputArgumentArray: (&protocol.MethodArgumentArrayBuilder{}).Build(),
			CallResult:          protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
		}, err
	}

	// check permissions
	err = s.verifyMethodPermissions(contractInfo, methodInfo, input.CallingService, input.CallingPermissionScope, input.AccessScope)
	if err != nil {
		return &services.ProcessCallOutput{
			OutputArgumentArray: (&protocol.MethodArgumentArrayBuilder{}).Build(),
			CallResult:          protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
		}, err
	}

	// execute
	outputArgs, contractErr, err := s.processMethodCall(executionContextId, contractInfo, methodInfo, input.InputArgumentArray)
	if outputArgs == nil {
		outputArgs = (&protocol.MethodArgumentArrayBuilder{}).Build()
	}
	if err != nil {
		return &services.ProcessCallOutput{
			OutputArgumentArray: outputArgs,
			CallResult:          protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
		}, err
	}

	// result
	callResult := protocol.EXECUTION_RESULT_SUCCESS
	if contractErr != nil {
		callResult = protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT
	}
	return &services.ProcessCallOutput{
		OutputArgumentArray: outputArgs,
		CallResult:          callResult,
	}, contractErr
}

func (s *service) GetContractInfo(input *services.GetContractInfoInput) (*services.GetContractInfoOutput, error) {
	// retrieve code
	executionContextId := types.Context(input.ContextId)
	contractInfo, err := s.retrieveContractInfoFromRepository(executionContextId, input.ContractName)
	if err != nil {
		return nil, err
	}

	// result
	return &services.GetContractInfoOutput{
		PermissionScope: contractInfo.Permission,
	}, nil
}

func (s *service) getContractSdkHandler() handlers.ContractSdkCallHandler {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.contractSdkHandlerUnderMutex
}

func (s *service) getContractInstanceFromRepository(contractName primitives.ContractName) types.Contract {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.contractRepositoryUnderMutex == nil {
		return nil
	}
	return s.contractRepositoryUnderMutex[contractName]
}

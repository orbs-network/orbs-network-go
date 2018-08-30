package native

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository"
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
)

type service struct {
	reporting log.BasicLogger

	contractRepository map[primitives.ContractName]types.Contract
}

func NewNativeProcessor(
	reporting log.BasicLogger,
) services.Processor {
	return &service{
		reporting: reporting.For(log.Service("processor-native")),
	}
}

// runs once on system initialization (called by the virtual machine constructor)
func (s *service) RegisterContractSdkCallHandler(handler handlers.ContractSdkCallHandler) {
	s.contractRepository = make(map[primitives.ContractName]types.Contract)
	for _, contract := range repository.Contracts {
		s.contractRepository[contract.Name] = contract.InitSingleton(types.NewBaseContract(
			&stateSdk{handler, contract.Permission},
			&serviceSdk{handler, contract.Permission},
		))
	}
}

func (s *service) ProcessCall(input *services.ProcessCallInput) (*services.ProcessCallOutput, error) {
	if s.contractRepository == nil {
		return &services.ProcessCallOutput{
			OutputArguments: nil,
			CallResult:      protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
		}, errors.New("contractRepository is not initialized")
	}

	// retrieve code
	contractInfo, methodInfo, err := s.getContractAndMethodFromRepository(input.ContractName, input.MethodName)
	if err != nil {
		return &services.ProcessCallOutput{
			OutputArguments: nil,
			CallResult:      protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
		}, err
	}

	// check permissions
	err = s.verifyMethodPermissions(contractInfo, methodInfo, input.CallingService, input.CallingPermissionScope, input.AccessScope)
	if err != nil {
		return &services.ProcessCallOutput{
			OutputArguments: nil,
			CallResult:      protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
		}, err
	}

	// execute
	ctx := types.Context(input.ContextId)
	outputArgs, contractErr, err := s.processMethodCall(ctx, contractInfo, methodInfo, input.InputArguments)
	if err != nil {
		return &services.ProcessCallOutput{
			OutputArguments: nil,
			CallResult:      protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
		}, err
	}

	// result
	callResult := protocol.EXECUTION_RESULT_SUCCESS
	if contractErr != nil {
		callResult = protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT
	}
	return &services.ProcessCallOutput{
		OutputArguments: outputArgs,
		CallResult:      callResult,
	}, contractErr
}

func (s *service) GetContractInfo(input *services.GetContractInfoInput) (*services.GetContractInfoOutput, error) {
	if s.contractRepository == nil {
		return nil, errors.New("contractRepository is not initialized")
	}

	// retrieve code
	contractInfo, err := s.getContractFromRepository(input.ContractName)
	if err != nil {
		return nil, err
	}

	// result
	return &services.GetContractInfoOutput{
		PermissionScope: contractInfo.Permission,
	}, nil
}

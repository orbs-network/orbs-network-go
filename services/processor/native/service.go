package native

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository"
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
)

type service struct {
	contractRepository map[primitives.ContractName]types.Contract
}

func NewNativeProcessor() services.Processor {
	return &service{}
}

func (s *service) ProcessCall(input *services.ProcessCallInput) (*services.ProcessCallOutput, error) {
	if s.contractRepository == nil {
		return nil, errors.New("contractRepository is not initialized")
	}

	// retrieve code
	contractInfo, methodInfo, err := s.retrieveMethodFromRepository(input.ContractName, input.MethodName)
	if err != nil {
		return &services.ProcessCallOutput{
			OutputArguments: nil,
			CallResult:      protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
		}, err
	}

	// check permissions
	err = s.verifyMethodPermissions(contractInfo, methodInfo, input.CallingService, input.PermissionScope, input.AccessScope)
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

func (s *service) DeployNativeService(input *services.DeployNativeServiceInput) (*services.DeployNativeServiceOutput, error) {
	panic("Not implemented")
}

func (s *service) RegisterContractSdkCallHandler(handler handlers.ContractSdkCallHandler) {
	baseContract := types.NewBaseContract(
		&stateSdk{handler},
	)
	s.contractRepository = make(map[primitives.ContractName]types.Contract)
	for _, contract := range repository.Contracts {
		s.contractRepository[contract.Name] = contract.Implementation(baseContract)
	}
}

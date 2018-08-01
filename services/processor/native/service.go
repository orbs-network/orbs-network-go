package native

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

type service struct {
}

func NewNativeProcessor() services.Processor {
	return &service{}
}

func (s *service) ProcessCall(input *services.ProcessCallInput) (*services.ProcessCallOutput, error) {
	// retrieve code
	contractInfo, methodInfo, err := s.retrieveMethodFromRepository(input.ContractName, input.MethodName)
	if err != nil {
		return nil, err
	}

	// check permissions
	err = s.verifyMethodPermissions(contractInfo, methodInfo, input.CallingService, input.PermissionScope, input.AccessScope)
	if err != nil {
		return nil, err
	}

	// execute
	outputArgs, err := s.processMethodCall(contractInfo, methodInfo, input.InputArguments)
	if err != nil {
		return nil, err
	}

	return &services.ProcessCallOutput{
		OutputArguments: outputArgs,
		CallResult:      protocol.EXECUTION_RESULT_SUCCESS,
	}, nil
}

func (s *service) DeployNativeService(input *services.DeployNativeServiceInput) (*services.DeployNativeServiceOutput, error) {
	panic("Not implemented")
}

func (s *service) RegisterContractSdkCallHandler(handler handlers.ContractSdkCallHandler) {
	panic("Not implemented")
}

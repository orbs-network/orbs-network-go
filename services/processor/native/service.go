package native

import (
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

type service struct {
}

func NewNativeProcessor() services.Processor {
	return &service{}
}

func (s *service) ProcessCall(input *services.ProcessCallInput) (*services.ProcessCallOutput, error) {
	if input.ContractName == "" || input.MethodName == "" {
		panic("ProcessCall received corrupt args")
	}
	_, _, err := s.retrieveMethodFromRepository(input.ContractName, input.MethodName)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *service) DeployNativeService(input *services.DeployNativeServiceInput) (*services.DeployNativeServiceOutput, error) {
	panic("Not implemented")
}

func (s *service) RegisterContractSdkCallHandler(handler handlers.ContractSdkCallHandler) {
	panic("Not implemented")
}

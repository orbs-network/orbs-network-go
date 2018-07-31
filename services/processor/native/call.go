package native

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository"
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
)

func (s *service) retrieveMethodFromRepository(contractName primitives.ContractName, methodName primitives.MethodName) (*types.ContractInfo, *types.MethodInfo, error) {
	for _, contract := range repository.Contracts {
		if contractName.Equal(contract.Name) {
			for _, method := range contract.Methods {
				if methodName.Equal(method.Name) {
					return &contract, &method, nil
				}
			}
			return nil, nil, errors.Errorf("method '%s' not found in contract '%s'", methodName, contractName)
		}
	}
	return nil, nil, errors.Errorf("contract '%s' not found", contractName)
}

func (s *service) processMethodCall(*types.ContractInfo, *types.MethodInfo) error {
	return nil
}

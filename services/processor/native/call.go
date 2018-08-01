package native

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository"
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
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

func (s *service) verifyMethodPermissions(contractInfo *types.ContractInfo, methodInfo *types.MethodInfo, callingService primitives.ContractName, permissionScope protocol.ExecutionPermissionScope, accessScope protocol.ExecutionAccessScope) error {
	// allow external but protect internal
	if !methodInfo.External {
		err := s.verifyInternalMethodCall(contractInfo, methodInfo, callingService, permissionScope)
		if err != nil {
			return err
		}
	}

	// allow read but protect write
	if methodInfo.Access == protocol.ACCESS_SCOPE_READ_WRITE {
		if accessScope != protocol.ACCESS_SCOPE_READ_WRITE {
			return errors.Errorf("write method '%s' called without write access", methodInfo.Name)
		}
	}

	return nil
}

func (s *service) verifyInternalMethodCall(contractInfo *types.ContractInfo, methodInfo *types.MethodInfo, callingService primitives.ContractName, permissionScope protocol.ExecutionPermissionScope) error {
	if callingService.Equal(contractInfo.Name) {
		return nil
	}
	if permissionScope == protocol.PERMISSION_SCOPE_SYSTEM {
		return nil
	}
	return errors.Errorf("internal method '%s' called from different service '%s' without system permissions", methodInfo.Name, callingService)
}

func (s *service) processMethodCall(*types.ContractInfo, *types.MethodInfo) error {
	return nil
}

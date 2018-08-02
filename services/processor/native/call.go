package native

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository"
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"reflect"
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

func (s *service) processMethodCall(contractInfo *types.ContractInfo, methodInfo *types.MethodInfo, args []*protocol.MethodArgument) ([]*protocol.MethodArgument, error) {
	err := s.verifyMethodArgs(methodInfo, methodInfo.Implementation, args)
	if err != nil {
		return nil, err
	}
	return []*protocol.MethodArgument{}, nil
}

func (s *service) verifyMethodArgs(methodInfo *types.MethodInfo, implementation interface{}, args []*protocol.MethodArgument) error {
	methodType := reflect.ValueOf(implementation).Type()
	if methodType.NumIn()-1 != len(args) {
		return errors.Errorf("method '%s' takes %d args but received %d", methodInfo.Name, methodType.NumIn()-1, len(args))
	}
	for i := 1; i < methodType.NumIn(); i++ {
		switch methodType.In(i).Name() {
		case "uint32":
			if !args[i-1].IsTypeUint32Value() {
				return errors.Errorf("method '%s' expects arg %d to be uint32 but it has %s", methodInfo.Name, i-1, args[i-1].Type())
			}
		case "uint64":
			if !args[i-1].IsTypeUint64Value() {
				return errors.Errorf("method '%s' expects arg %d to be uint64 but it has %s", methodInfo.Name, i-1, args[i-1].Type())
			}
		case "string":
			if !args[i-1].IsTypeStringValue() {
				return errors.Errorf("method '%s' expects arg %d to be string but it has %s", methodInfo.Name, i-1, args[i-1].Type())
			}
		case "[]byte":
			if !args[i-1].IsTypeBytesValue() {
				return errors.Errorf("method '%s' expects arg %d to be bytes but it has %s", methodInfo.Name, i-1, args[i-1].Type())
			}
		}
	}
	return nil
}

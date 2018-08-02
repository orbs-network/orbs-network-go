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

func (s *service) processMethodCall(contractInfo *types.ContractInfo, methodInfo *types.MethodInfo, args []*protocol.MethodArgument) ([]*protocol.MethodArgument, error, error) {
	// verify input args
	values, err := s.verifyMethodArgs(methodInfo, methodInfo.Implementation, args)
	if err != nil {
		return nil, nil, err
	}

	// execute
	contractContextValue := reflect.ValueOf(s.contractRepository[contractInfo.Name])
	inValues := append([]reflect.Value{contractContextValue}, values...)
	outValues := reflect.ValueOf(methodInfo.Implementation).Call(inValues)
	if len(outValues) == 0 {
		return nil, nil, errors.Errorf("call method '%s' returned zero args although error is mandatory", methodInfo.Name)
	}

	// create output args
	outArgs := []*protocol.MethodArgument{}
	if len(outValues) > 1 {
		outArgs, err = s.createMethodArgs(methodInfo, outValues[:len(outValues)-1])
		if err != nil {
			return nil, nil, err
		}
	}

	// get contract error
	outErr, err := s.createContractError(methodInfo, outValues[len(outValues)-1])
	return outArgs, outErr, err
}

func (s *service) verifyMethodArgs(methodInfo *types.MethodInfo, implementation interface{}, args []*protocol.MethodArgument) ([]reflect.Value, error) {
	res := []reflect.Value{}
	methodType := reflect.ValueOf(implementation).Type()
	if methodType.NumIn()-1 != len(args) {
		return nil, errors.Errorf("method '%s' takes %d args but received %d", methodInfo.Name, methodType.NumIn()-1, len(args))
	}
	for i := 1; i < methodType.NumIn(); i++ {
		switch methodType.In(i).Kind() {
		case reflect.Uint32:
			if !args[i-1].IsTypeUint32Value() {
				return nil, errors.Errorf("method '%s' expects arg %d to be uint32 but it has %s", methodInfo.Name, i-1, args[i-1].Type())
			}
			res = append(res, reflect.ValueOf(args[i-1].Uint32Value()))
		case reflect.Uint64:
			if !args[i-1].IsTypeUint64Value() {
				return nil, errors.Errorf("method '%s' expects arg %d to be uint64 but it has %s", methodInfo.Name, i-1, args[i-1].Type())
			}
			res = append(res, reflect.ValueOf(args[i-1].Uint64Value()))
		case reflect.String:
			if !args[i-1].IsTypeStringValue() {
				return nil, errors.Errorf("method '%s' expects arg %d to be string but it has %s", methodInfo.Name, i-1, args[i-1].Type())
			}
			res = append(res, reflect.ValueOf(args[i-1].StringValue()))
		case reflect.Slice:
			if methodType.In(i).Elem().Kind() != reflect.Uint8 {
				return nil, errors.Errorf("method '%s' arg %d slice type is not byte", methodInfo.Name, i-1)
			}
			if !args[i-1].IsTypeBytesValue() {
				return nil, errors.Errorf("method '%s' expects arg %d to be bytes but it has %s", methodInfo.Name, i-1, args[i-1].Type())
			}
			res = append(res, reflect.ValueOf(args[i-1].BytesValue()))
		default:
			return nil, errors.Errorf("method '%s' expects arg %d to be unknown type", methodInfo.Name, i-1, args[i-1].Type())
		}
	}
	return res, nil
}

func (s *service) createMethodArgs(methodInfo *types.MethodInfo, args []reflect.Value) ([]*protocol.MethodArgument, error) {
	res := []*protocol.MethodArgument{}
	for i, arg := range args {
		switch arg.Kind() {
		case reflect.Uint32:
			res = append(res, (&protocol.MethodArgumentBuilder{Name: "uint32", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_32_VALUE, Uint32Value: arg.Interface().(uint32)}).Build())
		case reflect.Uint64:
			res = append(res, (&protocol.MethodArgumentBuilder{Name: "uint64", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE, Uint64Value: arg.Interface().(uint64)}).Build())
		case reflect.String:
			res = append(res, (&protocol.MethodArgumentBuilder{Name: "string", Type: protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE, StringValue: arg.Interface().(string)}).Build())
		case reflect.Slice:
			if arg.Type().Elem().Kind() != reflect.Uint8 {
				return nil, errors.Errorf("call method '%s' output arg %d slice type is not byte", methodInfo.Name, i)
			}
			res = append(res, (&protocol.MethodArgumentBuilder{Name: "bytes", Type: protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE, BytesValue: arg.Interface().([]byte)}).Build())
		default:
			return nil, errors.Errorf("call method '%s' output arg %d is of unknown type", methodInfo.Name, i)
		}
	}
	return res, nil
}

func (s *service) createContractError(methodInfo *types.MethodInfo, value reflect.Value) (outErr error, err error) {
	if value.Interface() != nil {
		var ok bool
		outErr, ok = value.Interface().(error)
		if !ok {
			return nil, errors.Errorf("call method '%s' last arg returned is not a valid error", methodInfo.Name)
		}
		return outErr, nil
	}
	return nil, nil
}

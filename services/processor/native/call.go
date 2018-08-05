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

func (s *service) processMethodCall(ctx types.Context, contractInfo *types.ContractInfo, methodInfo *types.MethodInfo, args []*protocol.MethodArgument) (outArgs []*protocol.MethodArgument, outErr error, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("call method '%s' panicked: %v", methodInfo.Name, r)
		}
	}()

	// verify input args
	argValues, err := s.prepareMethodInputArgsForCall(ctx, methodInfo, methodInfo.Implementation, args)
	if err != nil {
		return nil, nil, err
	}

	// execute the call
	contractValue := reflect.ValueOf(s.contractRepository[contractInfo.Name])
	contextValue := reflect.ValueOf(ctx)
	inValues := append([]reflect.Value{contractValue, contextValue}, argValues...)
	outValues := reflect.ValueOf(methodInfo.Implementation).Call(inValues)
	if len(outValues) == 0 {
		return nil, nil, errors.Errorf("call method '%s' returned zero args although error is mandatory", methodInfo.Name)
	}

	// create output args
	outArgs = []*protocol.MethodArgument{}
	if len(outValues) > 1 {
		outArgs, err = s.createMethodOutputArgs(methodInfo, outValues[:len(outValues)-1])
		if err != nil {
			return nil, nil, err
		}
	}

	// create contract output error
	outErr, err = s.createContractOutputError(methodInfo, outValues[len(outValues)-1])
	return outArgs, outErr, err
}

func (s *service) prepareMethodInputArgsForCall(ctx types.Context, methodInfo *types.MethodInfo, implementation interface{}, args []*protocol.MethodArgument) ([]reflect.Value, error) {
	const NUM_ARGS_RECEIVER_AND_CONTEXT = 2

	res := []reflect.Value{}
	methodType := reflect.ValueOf(implementation).Type()
	if methodType.NumIn() < NUM_ARGS_RECEIVER_AND_CONTEXT || methodType.In(1) != reflect.TypeOf(ctx) {
		return nil, errors.Errorf("method '%s' first arg is not Context", methodInfo.Name)
	}

	if methodType.NumIn()-NUM_ARGS_RECEIVER_AND_CONTEXT != len(args) {
		return nil, errors.Errorf("method '%s' takes %d args but received %d", methodInfo.Name, methodType.NumIn()-NUM_ARGS_RECEIVER_AND_CONTEXT, len(args))
	}

	for i := 0; i < methodType.NumIn()-NUM_ARGS_RECEIVER_AND_CONTEXT; i++ {
		switch methodType.In(i + NUM_ARGS_RECEIVER_AND_CONTEXT).Kind() {
		case reflect.Uint32:
			if !args[i].IsTypeUint32Value() {
				return nil, errors.Errorf("method '%s' expects arg %d to be uint32 but it has %s", methodInfo.Name, i, args[i].Type())
			}
			res = append(res, reflect.ValueOf(args[i].Uint32Value()))
		case reflect.Uint64:
			if !args[i].IsTypeUint64Value() {
				return nil, errors.Errorf("method '%s' expects arg %d to be uint64 but it has %s", methodInfo.Name, i, args[i].Type())
			}
			res = append(res, reflect.ValueOf(args[i].Uint64Value()))
		case reflect.String:
			if !args[i].IsTypeStringValue() {
				return nil, errors.Errorf("method '%s' expects arg %d to be string but it has %s", methodInfo.Name, i, args[i].Type())
			}
			res = append(res, reflect.ValueOf(args[i].StringValue()))
		case reflect.Slice:
			if methodType.In(i+NUM_ARGS_RECEIVER_AND_CONTEXT).Elem().Kind() != reflect.Uint8 {
				return nil, errors.Errorf("method '%s' arg %d slice type is not byte", methodInfo.Name, i)
			}
			if !args[i].IsTypeBytesValue() {
				return nil, errors.Errorf("method '%s' expects arg %d to be bytes but it has %s", methodInfo.Name, i, args[i].Type())
			}
			res = append(res, reflect.ValueOf(args[i].BytesValue()))
		default:
			return nil, errors.Errorf("method '%s' expects arg %d to be unknown type", methodInfo.Name, i, args[i].Type())
		}
	}

	return res, nil
}

func (s *service) createMethodOutputArgs(methodInfo *types.MethodInfo, args []reflect.Value) ([]*protocol.MethodArgument, error) {
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

func (s *service) createContractOutputError(methodInfo *types.MethodInfo, value reflect.Value) (outErr error, err error) {
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

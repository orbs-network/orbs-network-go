package native

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"reflect"
)

func (s *service) verifyMethodPermissions(contractInfo *sdk.ContractInfo, methodInfo *sdk.MethodInfo, callingService primitives.ContractName, permissionScope protocol.ExecutionPermissionScope, accessScope protocol.ExecutionAccessScope) error {
	// allow external but protect internal
	if !methodInfo.External {
		err := s.verifyInternalMethodCall(contractInfo, methodInfo, callingService, permissionScope)
		if err != nil {
			return err
		}
	}

	// allow read but protect write
	if methodInfo.Access == sdk.ACCESS_SCOPE_READ_WRITE {
		if accessScope != protocol.ACCESS_SCOPE_READ_WRITE {
			return errors.Errorf("write method '%s' called without write access", methodInfo.Name)
		}
	}

	return nil
}

func (s *service) verifyInternalMethodCall(contractInfo *sdk.ContractInfo, methodInfo *sdk.MethodInfo, callingService primitives.ContractName, permissionScope protocol.ExecutionPermissionScope) error {
	if callingService.Equal(primitives.ContractName(contractInfo.Name)) {
		return nil
	}
	if permissionScope == protocol.PERMISSION_SCOPE_SYSTEM {
		return nil
	}
	return errors.Errorf("internal method '%s' called from different service '%s' without system permissions", methodInfo.Name, callingService)
}

func (s *service) processMethodCall(executionContextId sdk.Context, contractInfo *sdk.ContractInfo, methodInfo *sdk.MethodInfo, args *protocol.MethodArgumentArray) (contractOutputArgs *protocol.MethodArgumentArray, contractOutputErr error, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("call method '%s' panicked: %v", methodInfo.Name, r)
		}
	}()

	// verify input args
	argValues, err := s.prepareMethodInputArgsForCall(executionContextId, methodInfo, methodInfo.Implementation, args)
	if err != nil {
		return nil, nil, err
	}

	// execute the call
	s.reporting.Info("processor executing contract", log.String("contract", contractInfo.Name), log.String("method", methodInfo.Name))
	contractInstance := s.getContractInstanceFromRepository(contractInfo.Name)
	if contractInstance == nil {
		return nil, nil, errors.New("contract repository is not initialized yet")
	}
	contractValue := reflect.ValueOf(contractInstance)
	contextValue := reflect.ValueOf(executionContextId)
	inValues := append([]reflect.Value{contractValue, contextValue}, argValues...)
	outValues := reflect.ValueOf(methodInfo.Implementation).Call(inValues)
	if len(outValues) == 0 {
		return nil, nil, errors.Errorf("call method '%s' returned zero args although error is mandatory", methodInfo.Name)
	}

	// create output args
	contractOutputArgs = nil
	if len(outValues) > 1 {
		contractOutputArgs, err = s.createMethodOutputArgs(methodInfo, outValues[:len(outValues)-1])
		if err != nil {
			return nil, nil, err
		}
	}

	// create contract output error
	contractOutputErr, err = s.createContractOutputError(methodInfo, outValues[len(outValues)-1])
	return contractOutputArgs, contractOutputErr, err
}

func (s *service) prepareMethodInputArgsForCall(executionContextId sdk.Context, methodInfo *sdk.MethodInfo, implementation interface{}, args *protocol.MethodArgumentArray) ([]reflect.Value, error) {
	const NUM_ARGS_RECEIVER_AND_CONTEXT = 2

	res := []reflect.Value{}
	methodType := reflect.ValueOf(implementation).Type()
	if methodType.NumIn() < NUM_ARGS_RECEIVER_AND_CONTEXT || methodType.In(1) != reflect.TypeOf(executionContextId) {
		return nil, errors.Errorf("method '%s' first arg is not Context", methodInfo.Name)
	}

	var arg *protocol.MethodArgument
	argsIterator := args.ArgumentsIterator()
	for i := 0; i < methodType.NumIn()-NUM_ARGS_RECEIVER_AND_CONTEXT; i++ {

		// get the next arg from the transaction
		if argsIterator.HasNext() {
			arg = argsIterator.NextArguments()
		} else {
			return nil, errors.Errorf("method '%s' takes %d args but received %d", methodInfo.Name, methodType.NumIn()-NUM_ARGS_RECEIVER_AND_CONTEXT, i)
		}

		// translate argument type
		switch methodType.In(i + NUM_ARGS_RECEIVER_AND_CONTEXT).Kind() {
		case reflect.Uint32:
			if !arg.IsTypeUint32Value() {
				return nil, errors.Errorf("method '%s' expects arg %d to be uint32 but it has %s", methodInfo.Name, i, arg.StringType())
			}
			res = append(res, reflect.ValueOf(arg.Uint32Value()))
		case reflect.Uint64:
			if !arg.IsTypeUint64Value() {
				return nil, errors.Errorf("method '%s' expects arg %d to be uint64 but it has %s", methodInfo.Name, i, arg.StringType())
			}
			res = append(res, reflect.ValueOf(arg.Uint64Value()))
		case reflect.String:
			if !arg.IsTypeStringValue() {
				return nil, errors.Errorf("method '%s' expects arg %d to be string but it has %s", methodInfo.Name, i, arg.StringType())
			}
			res = append(res, reflect.ValueOf(arg.StringValue()))
		case reflect.Slice:
			if methodType.In(i+NUM_ARGS_RECEIVER_AND_CONTEXT).Elem().Kind() != reflect.Uint8 {
				return nil, errors.Errorf("method '%s' arg %d slice type is not byte", methodInfo.Name, i)
			}
			if !arg.IsTypeBytesValue() {
				return nil, errors.Errorf("method '%s' expects arg %d to be bytes but it has %s", methodInfo.Name, i, arg.StringType())
			}
			res = append(res, reflect.ValueOf(arg.BytesValue()))
		default:
			return nil, errors.Errorf("method '%s' expects arg %d to be a known type but it has %s", methodInfo.Name, i, arg.StringType())
		}

	}

	// make sure transaction doesn't have any more args left
	if argsIterator.HasNext() {
		return nil, errors.Errorf("method '%s' takes %d args but received more", methodInfo.Name, methodType.NumIn()-NUM_ARGS_RECEIVER_AND_CONTEXT)
	}

	return res, nil
}

func (s *service) createMethodOutputArgs(methodInfo *sdk.MethodInfo, args []reflect.Value) (*protocol.MethodArgumentArray, error) {
	res := []*protocol.MethodArgumentBuilder{}
	for i, arg := range args {
		switch arg.Kind() {
		case reflect.Uint32:
			res = append(res, &protocol.MethodArgumentBuilder{Name: "uint32", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_32_VALUE, Uint32Value: arg.Interface().(uint32)})
		case reflect.Uint64:
			res = append(res, &protocol.MethodArgumentBuilder{Name: "uint64", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE, Uint64Value: arg.Interface().(uint64)})
		case reflect.String:
			res = append(res, &protocol.MethodArgumentBuilder{Name: "string", Type: protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE, StringValue: arg.Interface().(string)})
		case reflect.Slice:
			if arg.Type().Elem().Kind() != reflect.Uint8 {
				return nil, errors.Errorf("call method '%s' output arg %d slice type is not byte", methodInfo.Name, i)
			}
			res = append(res, &protocol.MethodArgumentBuilder{Name: "bytes", Type: protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE, BytesValue: arg.Interface().([]byte)})
		default:
			return nil, errors.Errorf("call method '%s' output arg %d is of unknown type", methodInfo.Name, i)
		}
	}
	return (&protocol.MethodArgumentArrayBuilder{
		Arguments: res,
	}).Build(), nil
}

func (s *service) createContractOutputError(methodInfo *sdk.MethodInfo, value reflect.Value) (outErr error, err error) {
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

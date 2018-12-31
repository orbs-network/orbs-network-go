package native

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"reflect"
)

func (s *service) retrieveContractAndMethodInstances(contractName string, methodName string, permissionScope protocol.ExecutionPermissionScope) (contractInstance *types.ContractInstance, methodInstance types.MethodInstance, err error) {
	contractInstance = s.getContractInstance(contractName)
	if contractInstance == nil {
		return nil, nil, errors.Errorf("contract instance not found for contract '%s'", contractName)
	}

	methodInstance, found := contractInstance.PublicMethods[methodName]
	if found {
		return contractInstance, methodInstance, nil
	}

	methodInstance, found = contractInstance.SystemMethods[methodName]
	if found {
		if permissionScope == protocol.PERMISSION_SCOPE_SYSTEM {
			return contractInstance, methodInstance, nil
		} else {
			return nil, nil, errors.Errorf("only system contracts can run method '%s'", methodName)
		}
	}

	return nil, nil, errors.Errorf("method '%s' not found on contract '%s'", methodName, contractName)
}

func (s *service) processMethodCall(executionContextId primitives.ExecutionContextId, contractInstance *types.ContractInstance, methodInstance types.MethodInstance, args *protocol.ArgumentArray) (contractOutputArgs *protocol.ArgumentArray, contractOutputErr error, err error) {

	defer func() {
		if r := recover(); r != nil {
			contractOutputErr = errors.Errorf("%s", r)
			contractOutputArgs = s.createMethodOutputArgsWithString(contractOutputErr.Error())
		}
	}()

	// verify input args
	inValues, err := s.prepareMethodInputArgsForCall(methodInstance, args)
	if err != nil {
		return nil, nil, err
	}

	// execute the call
	outValues := reflect.ValueOf(methodInstance).Call(inValues)

	// create output args
	contractOutputArgs, err = s.createMethodOutputArgs(methodInstance, outValues)
	if err != nil {
		return nil, nil, err
	}

	// done
	return contractOutputArgs, contractOutputErr, err
}

func (s *service) prepareMethodInputArgsForCall(methodInstance types.MethodInstance, args *protocol.ArgumentArray) ([]reflect.Value, error) {
	res := []reflect.Value{}
	methodType := reflect.ValueOf(methodInstance).Type()

	var arg *protocol.Argument
	argsIterator := args.ArgumentsIterator()
	for i := 0; i < methodType.NumIn(); i++ {

		// get the next arg from the transaction
		if argsIterator.HasNext() {
			arg = argsIterator.NextArguments()
		} else {
			return nil, errors.Errorf("method takes %d args but received %d", methodType.NumIn(), i)
		}

		// translate argument type
		switch methodType.In(i).Kind() {
		case reflect.Uint32:
			if !arg.IsTypeUint32Value() {
				return nil, errors.Errorf("method expects arg %d to be uint32 but it has %s", i, arg.StringType())
			}
			res = append(res, reflect.ValueOf(arg.Uint32Value()))
		case reflect.Uint64:
			if !arg.IsTypeUint64Value() {
				return nil, errors.Errorf("method expects arg %d to be uint64 but it has %s", i, arg.StringType())
			}
			res = append(res, reflect.ValueOf(arg.Uint64Value()))
		case reflect.String:
			if !arg.IsTypeStringValue() {
				return nil, errors.Errorf("method expects arg %d to be string but it has %s", i, arg.StringType())
			}
			res = append(res, reflect.ValueOf(arg.StringValue()))
		case reflect.Slice:
			if methodType.In(i).Elem().Kind() != reflect.Uint8 {
				return nil, errors.Errorf("method arg %d slice type is not byte", i)
			}
			if !arg.IsTypeBytesValue() {
				return nil, errors.Errorf("method expects arg %d to be bytes but it has %s", i, arg.StringType())
			}
			res = append(res, reflect.ValueOf(arg.BytesValue()))
		default:
			return nil, errors.Errorf("method expects arg %d to be a known type but it has %s", i, arg.StringType())
		}

	}

	// make sure transaction doesn't have any more args left
	if argsIterator.HasNext() {
		return nil, errors.Errorf("method takes %d args but received more", methodType.NumIn())
	}

	return res, nil
}

func (s *service) createMethodOutputArgs(methodInstance types.MethodInstance, args []reflect.Value) (*protocol.ArgumentArray, error) {
	res := []*protocol.ArgumentBuilder{}
	for i, arg := range args {
		k := arg.Kind()
		switch k {
		case reflect.Uint32:
			res = append(res, &protocol.ArgumentBuilder{Type: protocol.ARGUMENT_TYPE_UINT_32_VALUE, Uint32Value: arg.Interface().(uint32)})
		case reflect.Uint64:
			res = append(res, &protocol.ArgumentBuilder{Type: protocol.ARGUMENT_TYPE_UINT_64_VALUE, Uint64Value: arg.Interface().(uint64)})
		case reflect.String:
			res = append(res, &protocol.ArgumentBuilder{Type: protocol.ARGUMENT_TYPE_STRING_VALUE, StringValue: arg.Interface().(string)})
		case reflect.Slice:
			if arg.Type().Elem().Kind() != reflect.Uint8 {
				return nil, errors.Errorf("method output arg %d slice type is not byte", i)
			}
			res = append(res, &protocol.ArgumentBuilder{Type: protocol.ARGUMENT_TYPE_BYTES_VALUE, BytesValue: arg.Interface().([]byte)})
		default:
			return nil, errors.Errorf("method output arg %d is of unknown type", i)
		}
	}
	return (&protocol.ArgumentArrayBuilder{
		Arguments: res,
	}).Build(), nil
}

func (s *service) createMethodOutputArgsWithString(str string) *protocol.ArgumentArray {
	return (&protocol.ArgumentArrayBuilder{
		Arguments: []*protocol.ArgumentBuilder{
			{Type: protocol.ARGUMENT_TYPE_STRING_VALUE, StringValue: str},
		},
	}).Build()
}

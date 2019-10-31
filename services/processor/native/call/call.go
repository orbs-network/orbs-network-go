// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package call

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"math/big"
	"reflect"
)

var bigIntType = reflect.TypeOf(big.NewInt(0))

func PrepareMethodInputArgsForCall(methodInstance types.MethodInstance, args *protocol.ArgumentArray, functionNameForErrors string) ([]reflect.Value, error) {
	res := []reflect.Value{}
	methodType := reflect.ValueOf(methodInstance).Type()

	var arg *protocol.Argument
	i := 0
	argsIterator := args.ArgumentsIterator()
	for ; argsIterator.HasNext(); i++ {
		// get the next arg from the transaction
		if argsIterator.HasNext() {
			arg = argsIterator.NextArguments()
		} else {
			return nil, errors.Errorf("method '%s' takes %d args but received %d", functionNameForErrors, methodType.NumIn(), i)
		}

		// in case of variable length we take the last available index
		methodTypeIndex := i
		if methodTypeIndex >= methodType.NumIn() {
			if !methodType.IsVariadic() {
				return nil, errors.Errorf("method '%s' takes %d args but received more", functionNameForErrors, methodType.NumIn())
			}
			methodTypeIndex = methodType.NumIn() - 1
		}
		methodTypeIn := methodType.In(methodTypeIndex)

		// translate argument type
		switch methodTypeIn.Kind() {
		case reflect.Uint32:
			if !arg.IsTypeUint32Value() {
				return nil, errors.Errorf("method '%s' expects arg %d to be uint32 but it has %s", functionNameForErrors, i, arg.StringType())
			}
			res = append(res, reflect.ValueOf(arg.Uint32Value()))
		case reflect.Uint64:
			if !arg.IsTypeUint64Value() {
				return nil, errors.Errorf("method '%s' expects arg %d to be uint64 but it has %s", functionNameForErrors, i, arg.StringType())
			}
			res = append(res, reflect.ValueOf(arg.Uint64Value()))
		case reflect.String:
			if !arg.IsTypeStringValue() {
				return nil, errors.Errorf("method '%s' expects arg %d to be string but it has %s", functionNameForErrors, i, arg.StringType())
			}
			res = append(res, reflect.ValueOf(arg.StringValue()))
		case reflect.Array:
			if methodTypeIn.Elem().Kind() == reflect.Uint8 {
				if methodTypeIn.Len() == 20 {
					if !arg.IsTypeBytes20Value() {
						return nil, errors.Errorf("method '%s' expects arg %d to be [20]byte but it has %s", functionNameForErrors, i, arg.StringType())
					}
					res = append(res, reflect.ValueOf(arg.Bytes20Value()))
				} else if methodTypeIn.Len() == 32 {
					if !arg.IsTypeBytes32Value() {
						return nil, errors.Errorf("method '%s' expects arg %d to be [32]byte but it has %s", functionNameForErrors, i, arg.StringType())
					}
					res = append(res, reflect.ValueOf(arg.Bytes32Value()))
				} else {
					return nil, errors.Errorf("method '%s' expects arg %d to be a known type but it has %s", functionNameForErrors, i, arg.StringType())
				}
			} else {
				return nil, errors.Errorf("method '%s' expects arg %d to be a known type but it has %s", functionNameForErrors, i, arg.StringType())
			}
		case reflect.Bool:
			if !arg.IsTypeBoolValue() {
				return nil, errors.Errorf("method '%s' expects arg %d to be bool but it has %s", functionNameForErrors, i, arg.StringType())
			}
			res = append(res, reflect.ValueOf(arg.BoolValue()))
		case reflect.Ptr:
			if methodTypeIn == bigIntType {
				if !arg.IsTypeUint256Value() {
					return nil, errors.Errorf("method '%s' expects arg %d to be *big.Int but it has %s", functionNameForErrors, i, arg.StringType())
				}
				res = append(res, reflect.ValueOf(arg.Uint256Value()))
			} else {
				return nil, errors.Errorf("method '%s' expects arg %d to be a known type but it has %s", functionNameForErrors, i, arg.StringType())
			}
		case reflect.Slice:
			switch methodTypeIn.Elem().Kind() {
			case reflect.Uint8:
				if !arg.IsTypeBytesValue() {
					return nil, errors.Errorf("method '%s' expects arg %d to be []byte but it has %s", functionNameForErrors, i, arg.StringType())
				}
				res = append(res, reflect.ValueOf(arg.BytesValue()))
			case reflect.String:
				if methodType.IsVariadic() && !arg.IsTypeStringValue() {
					return nil, errors.Errorf("method '%s' expects arg %d to be string but it has %s", functionNameForErrors, i, arg.StringType())
				}
				res = append(res, reflect.ValueOf(arg.StringValue()))
			case reflect.Uint32:
				if methodType.IsVariadic() && !arg.IsTypeUint32Value() {
					return nil, errors.Errorf("method '%s' expects arg %d to be uint32 but it has %s", functionNameForErrors, i, arg.StringType())
				}
				res = append(res, reflect.ValueOf(arg.Uint32Value()))
			case reflect.Uint64:
				if methodType.IsVariadic() && !arg.IsTypeUint64Value() {
					return nil, errors.Errorf("method '%s' expects arg %d to be uint64 but it has %s", functionNameForErrors, i, arg.StringType())
				}
				res = append(res, reflect.ValueOf(arg.Uint64Value()))
			case reflect.Array:
				if methodTypeIn.Elem().Kind() == reflect.Uint8 {
					if methodTypeIn.Len() == 20 {
						if methodType.IsVariadic() && !arg.IsTypeBytes20Value() {
							return nil, errors.Errorf("method '%s' expects arg %d to be [20]byte but it has %s", functionNameForErrors, i, arg.StringType())
						}
						res = append(res, reflect.ValueOf(arg.Bytes20Value()))
					} else if methodTypeIn.Len() == 32 {
						if methodType.IsVariadic() && !arg.IsTypeBytes32Value() {
							return nil, errors.Errorf("method '%s' expects arg %d to be [32]byte but it has %s", functionNameForErrors, i, arg.StringType())
						}
						res = append(res, reflect.ValueOf(arg.Bytes32Value()))
					} else {
						return nil, errors.Errorf("method '%s' expects arg %d to be a known type but it has %s", functionNameForErrors, i, arg.StringType())
					}
				} else {
					return nil, errors.Errorf("method '%s' expects arg %d to be a known type but it has %s", functionNameForErrors, i, arg.StringType())
				}
			case reflect.Bool:
				if methodType.IsVariadic() && !arg.IsTypeBoolValue() {
					return nil, errors.Errorf("method '%s' expects arg %d to be bool but it has %s", functionNameForErrors, i, arg.StringType())
				}
				res = append(res, reflect.ValueOf(arg.BoolValue()))
			case reflect.Ptr:
				if methodTypeIn.Elem() == bigIntType {
					if methodType.IsVariadic() && !arg.IsTypeUint256Value() {
						return nil, errors.Errorf("method '%s' expects arg %d to be *big.Int but it has %s", functionNameForErrors, i, arg.StringType())
					}
					res = append(res, reflect.ValueOf(arg.Uint256Value()))
				} else {
					return nil, errors.Errorf("method '%s' expects arg %d to be a known type but it has %s", functionNameForErrors, i, arg.StringType())
				}
			case reflect.Slice:
				if methodType.IsVariadic() && (!arg.IsTypeBytesValue() ||
					(methodTypeIn.Elem().Elem().Kind() != reflect.Uint8)) { // check that element of slice-of-slice is defined as byte
					return nil, errors.Errorf("method '%s' expects arg %d to be [][]byte but it has %s", functionNameForErrors, i, arg.StringType())
				}
				res = append(res, reflect.ValueOf(arg.BytesValue()))
			default:
				return nil, errors.Errorf("method '%s' expect arg %d to be of different type", functionNameForErrors, i)
			}
		default:
			return nil, errors.Errorf("method '%s' expects arg %d to be a known type but it has %s", functionNameForErrors, i, arg.StringType())
		}
	}

	if methodType.IsVariadic() { // determine dangling array
		if i < methodType.NumIn()-1 { //
			return nil, errors.Errorf("method '%s' takes at least %d args but received less", functionNameForErrors, methodType.NumIn()-1)
		}
		return res, nil
	} else if i < methodType.NumIn() {
		return nil, errors.Errorf("method '%s' takes %d args but received less", functionNameForErrors, methodType.NumIn())
	}

	return res, nil
}

func CreateMethodOutputArgs(methodInstance types.MethodInstance, args []reflect.Value, functionNameForErrors string) (*protocol.ArgumentArray, error) {
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
		case reflect.Array:
			if arg.Type().Elem().Kind() == reflect.Uint8 {
				if arg.Len() == 20 {
					res = append(res, &protocol.ArgumentBuilder{Type: protocol.ARGUMENT_TYPE_BYTES_20_VALUE, Bytes20Value: arg.Interface().([20]byte)})
				} else if arg.Len() == 32 {
					res = append(res, &protocol.ArgumentBuilder{Type: protocol.ARGUMENT_TYPE_BYTES_32_VALUE, Bytes32Value: arg.Interface().([32]byte)})
				} else {
					return nil, errors.Errorf("method '%s' output arg %d is of unsupported type %s", functionNameForErrors, i, arg.Type())
				}
			} else {
				return nil, errors.Errorf("method '%s' output arg %d is of unsupported type %s", functionNameForErrors, i, arg.Type())
			}
		case reflect.Bool:
			res = append(res, &protocol.ArgumentBuilder{Type: protocol.ARGUMENT_TYPE_BOOL_VALUE, BoolValue: arg.Interface().(bool)})
		case reflect.Ptr:
			if arg.Type() == bigIntType {
				res = append(res, &protocol.ArgumentBuilder{Type: protocol.ARGUMENT_TYPE_UINT_256_VALUE, Uint256Value: arg.Interface().(*big.Int)})
			} else {
				return nil, errors.Errorf("method '%s' output arg %d is of unsupported type %s", functionNameForErrors, i, arg.Type())
			}
		case reflect.Slice:
			if arg.Type().Elem().Kind() != reflect.Uint8 {
				return nil, errors.Errorf("method '%s' output arg %d slice type is not byte", functionNameForErrors, i)
			}
			res = append(res, &protocol.ArgumentBuilder{Type: protocol.ARGUMENT_TYPE_BYTES_VALUE, BytesValue: arg.Interface().([]byte)})
		default:
			return nil, errors.Errorf("method '%s' output arg %d is of unsupported type %s", functionNameForErrors, i, arg.Type())
		}
	}
	return (&protocol.ArgumentArrayBuilder{
		Arguments: res,
	}).Build(), nil
}

// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package sdk

import (
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"reflect"
)

type SDKConfig interface {
	VirtualChainId() primitives.VirtualChainId
}

type service struct {
	sdkHandler handlers.ContractSdkCallHandler
	config     SDKConfig
}

func NewSDK(handler handlers.ContractSdkCallHandler) sdkContext.SdkHandler {
	return &service{
		sdkHandler: handler,
	}
}

func (s *service) prepareMethodInputArgsForCall(methodInstance types.MethodInstance, args *protocol.ArgumentArray, functionNameForErrors string) ([]reflect.Value, error) {
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
		return res, nil
	} else if i < methodType.NumIn() {
		return nil, errors.Errorf("method '%s' takes %d args but received less", functionNameForErrors, methodType.NumIn())
	} else if i > methodType.NumIn() {
		return nil, errors.Errorf("method '%s' takes %d args but received more", functionNameForErrors, methodType.NumIn())
	}

	return res, nil
}

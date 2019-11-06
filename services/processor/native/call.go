// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package native

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"reflect"
)

func processMethodCall(executionContextId primitives.ExecutionContextId, contractInstance *types.ContractInstance, methodInstance types.MethodInstance, args *protocol.ArgumentArray, functionNameForErrors string) (contractOutputArgs *protocol.ArgumentArray, contractOutputErr error, err error) {

	defer func() {
		if r := recover(); r != nil {
			contractOutputErr = errors.Errorf("%s", r)
			contractOutputArgs = createMethodOutputArgsWithString(contractOutputErr.Error())
		}
	}()

	// translate input protocol.Arguments to native
	inArgs, err := args.ToNatives()
	if err != nil {
		return nil, nil, err
	}

	// verify input args
	inValues, err := verifyMethodInputArgs(methodInstance, functionNameForErrors, inArgs)
	if err != nil {
		return nil, nil, err
	}

	// execute the call
	outValues := reflect.ValueOf(methodInstance).Call(inValues)

	// create output args
	contractOutputArgs, err = createMethodOutputArgs(outValues, functionNameForErrors)
	if err != nil {
		return nil, nil, err
	}

	// done
	return contractOutputArgs, contractOutputErr, err
}

func verifyMethodInputArgs(methodInstance types.MethodInstance, functionNameForErrors string, args []interface{}) ([]reflect.Value, error) {
	var res []reflect.Value
	methodType := reflect.ValueOf(methodInstance).Type()

	numOfArgs := len(args)
	indexOfLastArg := methodType.NumIn() - 1
	// check size of input vs method requirement
	if methodType.IsVariadic() { // determine dangling array
		if numOfArgs < indexOfLastArg {
			return nil, errors.Errorf("method '%s' takes at least %d args but received less", functionNameForErrors, indexOfLastArg)
		}
	} else if numOfArgs < methodType.NumIn() {
		return nil, errors.Errorf("method '%s' takes %d args but received less", functionNameForErrors, methodType.NumIn())
	} else if numOfArgs > methodType.NumIn() {
		return nil, errors.Errorf("method '%s' takes %d args but received more", functionNameForErrors, methodType.NumIn())
	}

	for i := 0 ; i < numOfArgs; i++ {
		argType := reflect.TypeOf(args[i])
		if methodType.IsVariadic() && i >= indexOfLastArg {
			typeOfVariadicArg := methodType.In(indexOfLastArg).Elem()
			if argType != typeOfVariadicArg {
				return nil, errors.Errorf("method '%s' expects arg %d to be %s but it has %s", functionNameForErrors, i, typeOfVariadicArg, argType)
			}
		} else if argType != methodType.In(i) {
			return nil, errors.Errorf("method '%s' expects arg %d to be %s but it has %s", functionNameForErrors, i, methodType.In(i), argType)
		}
		res = append(res, reflect.ValueOf(args[i]))
	}

	return res, nil
}

func createMethodOutputArgs(args []reflect.Value, functionNameForErrors string) (*protocol.ArgumentArray, error) {
	var argInterfaces []interface{}
	for _, arg := range args {
		argInterfaces = append(argInterfaces, arg.Interface())
	}

	res, err := protocol.ArgumentArrayFromNatives(argInterfaces)
	if err != nil {
		return nil, errors.Errorf("method '%s' output %s", functionNameForErrors, err.Error())
	}

	return res, nil
}

func createMethodOutputArgsWithString(str string) *protocol.ArgumentArray {
	res, _ :=  protocol.ArgumentArrayFromNatives([]interface{}{str})  // err ignored because we support argument with type string
	return res
}

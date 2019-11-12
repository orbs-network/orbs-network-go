// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package sdk

import (
	"context"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"reflect"
)

const SDK_OPERATION_NAME_EVENTS = "Sdk.Events"

func (s *service) SdkEventsEmitEvent(executionContextId sdkContext.ContextId, permissionScope sdkContext.PermissionScope, eventFunctionSignature interface{}, args ...interface{}) {
	eventName, err := types.GetContractMethodNameFromFunction(eventFunctionSignature)
	if err != nil {
		panic(errors.Wrapf(err, "failed to find event signature function"))
	}

	// verify event arguments are allowed to be packed and match signature
	eventArguments, err := protocol.ArgumentArrayFromNatives(args)
	if err != nil {
		panic(errors.Errorf("event '%s' input arguments: %s", eventName, err))
	}
	_, err = verifyEventMethodInputArgs(eventFunctionSignature, args)
	if err != nil {
		panic(errors.Errorf("event '%s' %s", eventName, err))
	}

	_, err = s.sdkHandler.HandleSdkCall(context.TODO(), &handlers.HandleSdkCallInput{
		ContextId:     primitives.ExecutionContextId(executionContextId),
		OperationName: SDK_OPERATION_NAME_EVENTS,
		MethodName:    "emitEvent",
		InputArguments: []*protocol.Argument{
			(&protocol.ArgumentBuilder{
				// eventName
				Type:        protocol.ARGUMENT_TYPE_STRING_VALUE,
				StringValue: eventName,
			}).Build(),
			(&protocol.ArgumentBuilder{
				// inputArgs
				Type:       protocol.ARGUMENT_TYPE_BYTES_VALUE,
				BytesValue: eventArguments.Raw(),
			}).Build(),
		},
		PermissionScope: protocol.ExecutionPermissionScope(permissionScope),
	})
	if err != nil {
		panic(errors.Wrapf(err, "failed to emit event '%s'", eventName))
	}
}

func verifyEventMethodInputArgs(eventSignature types.MethodInstance, args []interface{}) ([]reflect.Value, error) {
	var res []reflect.Value
	methodType := reflect.ValueOf(eventSignature).Type()
	if methodType.IsVariadic() { // determine dangling array
		return nil, errors.Errorf("is not allowed to be variadic")
	}

	numOfArgs := len(args)
	if numOfArgs != methodType.NumIn() {
		return nil, errors.Errorf("takes %d args but received %d", methodType.NumIn(), numOfArgs)
	}

	for i := 0; i < numOfArgs; i++ {
		argType := reflect.TypeOf(args[i])
		if argType != methodType.In(i) {
			return nil, errors.Errorf("expects arg %d to be %s but it has %s", i, methodType.In(i), argType)
		}
		res = append(res, reflect.ValueOf(args[i]))
	}

	return res, nil
}

// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package native

import (
	"context"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

const SDK_OPERATION_NAME_SERVICE = "Sdk.Service"

func (s *service) SdkServiceCallMethod(executionContextId sdkContext.ContextId, permissionScope sdkContext.PermissionScope, serviceName string, methodName string, args ...interface{}) []interface{} {
	output, err := s.sdkHandler.HandleSdkCall(context.TODO(), &handlers.HandleSdkCallInput{
		ContextId:     primitives.ExecutionContextId(executionContextId),
		OperationName: SDK_OPERATION_NAME_SERVICE,
		MethodName:    "callMethod",
		InputArguments: []*protocol.Argument{
			(&protocol.ArgumentBuilder{
				// serviceName
				Type:        protocol.ARGUMENT_TYPE_STRING_VALUE,
				StringValue: serviceName,
			}).Build(),
			(&protocol.ArgumentBuilder{
				// methodName
				Type:        protocol.ARGUMENT_TYPE_STRING_VALUE,
				StringValue: methodName,
			}).Build(),
			(&protocol.ArgumentBuilder{
				// inputArgs
				Type:       protocol.ARGUMENT_TYPE_BYTES_VALUE,
				BytesValue: argsToArgumentArray(args...).Raw(),
			}).Build(),
		},
		PermissionScope: protocol.ExecutionPermissionScope(permissionScope),
	})
	if err != nil {
		panic(err.Error())
	}
	if len(output.OutputArguments) != 1 || !output.OutputArguments[0].IsTypeBytesValue() {
		panic("callMethod Sdk.Service returned corrupt output value")
	}
	ArgumentArray := protocol.ArgumentArrayReader(output.OutputArguments[0].BytesValue())
	return ArgumentArrayToArgs(ArgumentArray)
}

func argsToArgumentArray(args ...interface{}) *protocol.ArgumentArray {
	res := []*protocol.ArgumentBuilder{}
	for _, arg := range args {
		switch arg.(type) {
		case uint32:
			res = append(res, &protocol.ArgumentBuilder{Type: protocol.ARGUMENT_TYPE_UINT_32_VALUE, Uint32Value: arg.(uint32)})
		case uint64:
			res = append(res, &protocol.ArgumentBuilder{Type: protocol.ARGUMENT_TYPE_UINT_64_VALUE, Uint64Value: arg.(uint64)})
		case string:
			res = append(res, &protocol.ArgumentBuilder{Type: protocol.ARGUMENT_TYPE_STRING_VALUE, StringValue: arg.(string)})
		case []byte:
			res = append(res, &protocol.ArgumentBuilder{Type: protocol.ARGUMENT_TYPE_BYTES_VALUE, BytesValue: arg.([]byte)})
		}
	}
	return (&protocol.ArgumentArrayBuilder{Arguments: res}).Build()
}

func ArgumentArrayToArgs(ArgumentArray *protocol.ArgumentArray) []interface{} {
	res := []interface{}{}
	for i := ArgumentArray.ArgumentsIterator(); i.HasNext(); {
		Argument := i.NextArguments()
		switch Argument.Type() {
		case protocol.ARGUMENT_TYPE_UINT_32_VALUE:
			res = append(res, Argument.Uint32Value())
		case protocol.ARGUMENT_TYPE_UINT_64_VALUE:
			res = append(res, Argument.Uint64Value())
		case protocol.ARGUMENT_TYPE_STRING_VALUE:
			res = append(res, Argument.StringValue())
		case protocol.ARGUMENT_TYPE_BYTES_VALUE:
			res = append(res, Argument.BytesValue())
		}
	}
	return res
}

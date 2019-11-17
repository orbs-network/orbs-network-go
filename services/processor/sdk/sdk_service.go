// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package sdk

import (
	"context"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
)

const SDK_OPERATION_NAME_SERVICE = "Sdk.Service"

func (s *service) SdkServiceCallMethod(executionContextId sdkContext.ContextId, permissionScope sdkContext.PermissionScope, serviceName string, methodName string, args ...interface{}) []interface{} {
	inputArguments, err := protocol.ArgumentArrayFromNatives(args)
	if err != nil {
		panic(errors.Wrap(err, "input arguments"))
	}

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
				BytesValue: inputArguments.Raw(),
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
	nativeArgumentsArray, err := protocol.ArgumentArrayReader(output.OutputArguments[0].BytesValue()).ToNatives()
	if err != nil {
		panic(errors.Wrap(err, "output arguments"))
	}
	return nativeArgumentsArray
}

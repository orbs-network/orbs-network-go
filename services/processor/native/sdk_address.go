// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package native

import (
	"context"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

const SDK_OPERATION_NAME_ADDRESS = "Sdk.Address"

// TODO(https://github.com/orbs-network/orbs-network-go/issues/584): fix context here
func (s *service) SdkAddressGetSignerAddress(executionContextId sdkContext.ContextId, permissionScope sdkContext.PermissionScope) []byte {
	output, err := s.sdkHandler.HandleSdkCall(context.TODO(), &handlers.HandleSdkCallInput{
		ContextId:       primitives.ExecutionContextId(executionContextId),
		OperationName:   SDK_OPERATION_NAME_ADDRESS,
		MethodName:      "getSignerAddress",
		InputArguments:  []*protocol.Argument{},
		PermissionScope: protocol.ExecutionPermissionScope(permissionScope),
	})
	if err != nil {
		panic(err.Error())
	}
	if len(output.OutputArguments) != 1 || !output.OutputArguments[0].IsTypeBytesValue() {
		panic("getSignerAddress Sdk.Address returned corrupt output value")
	}
	return output.OutputArguments[0].BytesValue()
}

func (s *service) SdkAddressGetCallerAddress(executionContextId sdkContext.ContextId, permissionScope sdkContext.PermissionScope) []byte {
	output, err := s.sdkHandler.HandleSdkCall(context.TODO(), &handlers.HandleSdkCallInput{
		ContextId:       primitives.ExecutionContextId(executionContextId),
		OperationName:   SDK_OPERATION_NAME_ADDRESS,
		MethodName:      "getCallerAddress",
		InputArguments:  []*protocol.Argument{},
		PermissionScope: protocol.ExecutionPermissionScope(permissionScope),
	})
	if err != nil {
		panic(err.Error())
	}
	if len(output.OutputArguments) != 1 || !output.OutputArguments[0].IsTypeBytesValue() {
		panic("getCallerAddress Sdk.Address returned corrupt output value")
	}
	return output.OutputArguments[0].BytesValue()
}

func (s *service) SdkAddressGetOwnAddress(executionContextId sdkContext.ContextId, permissionScope sdkContext.PermissionScope) []byte {
	output, err := s.sdkHandler.HandleSdkCall(context.TODO(), &handlers.HandleSdkCallInput{
		ContextId:       primitives.ExecutionContextId(executionContextId),
		OperationName:   SDK_OPERATION_NAME_ADDRESS,
		MethodName:      "getOwnAddress",
		InputArguments:  []*protocol.Argument{},
		PermissionScope: protocol.ExecutionPermissionScope(permissionScope),
	})
	if err != nil {
		panic(err.Error())
	}
	if len(output.OutputArguments) != 1 || !output.OutputArguments[0].IsTypeBytesValue() {
		panic("getOwnAddress Sdk.Address returned corrupt output value")
	}
	return output.OutputArguments[0].BytesValue()
}

func (s *service) SdkAddressGetContractAddress(executionContextId sdkContext.ContextId, permissionScope sdkContext.PermissionScope, contractName string) []byte {
	address, err := digest.CalcClientAddressOfContract(primitives.ContractName(contractName))
	if err != nil {
		panic(err.Error())
	}
	return address
}

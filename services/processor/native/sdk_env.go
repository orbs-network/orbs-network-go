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

const SDK_OPERATION_NAME_ENV = "Sdk.Env"

func (s *service) SdkEnvGetBlockHeight(executionContextId sdkContext.ContextId, permissionScope sdkContext.PermissionScope) uint64 {
	output, err := s.sdkHandler.HandleSdkCall(context.TODO(), &handlers.HandleSdkCallInput{
		ContextId:       primitives.ExecutionContextId(executionContextId),
		OperationName:   SDK_OPERATION_NAME_ENV,
		MethodName:      "getBlockHeight",
		InputArguments:  []*protocol.Argument{},
		PermissionScope: protocol.ExecutionPermissionScope(permissionScope),
	})
	if err != nil {
		panic(err.Error())
	}
	if len(output.OutputArguments) != 1 || !output.OutputArguments[0].IsTypeUint64Value() {
		panic("getBlockHeight Sdk.Env returned corrupt output value")
	}
	return output.OutputArguments[0].Uint64Value()
}

func (s *service) SdkEnvGetBlockTimestamp(executionContextId sdkContext.ContextId, permissionScope sdkContext.PermissionScope) uint64 {
	output, err := s.sdkHandler.HandleSdkCall(context.TODO(), &handlers.HandleSdkCallInput{
		ContextId:       primitives.ExecutionContextId(executionContextId),
		OperationName:   SDK_OPERATION_NAME_ENV,
		MethodName:      "getBlockTimestamp",
		InputArguments:  []*protocol.Argument{},
		PermissionScope: protocol.ExecutionPermissionScope(permissionScope),
	})
	if err != nil {
		panic(err.Error())
	}
	if len(output.OutputArguments) != 1 || !output.OutputArguments[0].IsTypeUint64Value() {
		panic("getBlockHeight Sdk.Env returned corrupt output value")
	}
	return output.OutputArguments[0].Uint64Value()
}

func (s *service) SdkEnvGetVirtualChainId(executionContextId sdkContext.ContextId, permissionScope sdkContext.PermissionScope) uint32 {
	return uint32(s.config.VirtualChainId())
}

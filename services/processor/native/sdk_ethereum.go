// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package native

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"strings"
)

const SDK_OPERATION_NAME_ETHEREUM = "Sdk.Ethereum"

func (s *service) SdkEthereumCallMethod(executionContextId sdkContext.ContextId, permissionScope sdkContext.PermissionScope, ethContractAddress string, jsonAbi string, ethBlockNumber uint64, methodName string, out interface{}, args ...interface{}) {
	parsedABI, err := abi.JSON(strings.NewReader(jsonAbi))
	if err != nil {
		panic(err.Error())
	}

	packedInput, err := ethereum.ABIPackFunctionInputArguments(parsedABI, methodName, args)
	if err != nil {
		panic(err.Error())
	}

	output, err := s.sdkHandler.HandleSdkCall(context.TODO(), &handlers.HandleSdkCallInput{
		ContextId:     primitives.ExecutionContextId(executionContextId),
		OperationName: SDK_OPERATION_NAME_ETHEREUM,
		MethodName:    "callMethod",
		InputArguments: []*protocol.Argument{
			(&protocol.ArgumentBuilder{
				// ethContractAddress
				Type:        protocol.ARGUMENT_TYPE_STRING_VALUE,
				StringValue: ethContractAddress,
			}).Build(),
			(&protocol.ArgumentBuilder{
				// jsonAbi
				Type:        protocol.ARGUMENT_TYPE_STRING_VALUE,
				StringValue: jsonAbi,
			}).Build(),
			(&protocol.ArgumentBuilder{
				// ethBlockNumber
				Type:        protocol.ARGUMENT_TYPE_UINT_64_VALUE,
				Uint64Value: ethBlockNumber,
			}).Build(),
			(&protocol.ArgumentBuilder{
				// methodName
				Type:        protocol.ARGUMENT_TYPE_STRING_VALUE,
				StringValue: methodName,
			}).Build(),
			(&protocol.ArgumentBuilder{
				// ethereumPackedInputArguments
				Type:       protocol.ARGUMENT_TYPE_BYTES_VALUE,
				BytesValue: packedInput,
			}).Build(),
		},
		PermissionScope: protocol.ExecutionPermissionScope(permissionScope),
	})
	if err != nil {
		panic(err.Error())
	}
	if len(output.OutputArguments) != 1 || !output.OutputArguments[0].IsTypeBytesValue() {
		panic("callMethod Sdk.Ethereum returned corrupt output value")
	}

	err = ethereum.ABIUnpackFunctionOutputArguments(parsedABI, out, methodName, output.OutputArguments[0].BytesValue())
	if err != nil {
		panic(err.Error())
	}
}

func (s *service) SdkEthereumGetTransactionLog(executionContextId sdkContext.ContextId, permissionScope sdkContext.PermissionScope, ethContractAddress string, jsonAbi string, ethTxHash string, eventName string, out interface{}) (ethBlockNumber uint64, ethTxIndex uint32) {
	parsedABI, err := abi.JSON(strings.NewReader(jsonAbi))
	if err != nil {
		panic(err.Error())
	}

	output, err := s.sdkHandler.HandleSdkCall(context.TODO(), &handlers.HandleSdkCallInput{
		ContextId:     primitives.ExecutionContextId(executionContextId),
		OperationName: SDK_OPERATION_NAME_ETHEREUM,
		MethodName:    "getTransactionLog",
		InputArguments: []*protocol.Argument{
			(&protocol.ArgumentBuilder{
				// ethContractAddress
				Type:        protocol.ARGUMENT_TYPE_STRING_VALUE,
				StringValue: ethContractAddress,
			}).Build(),
			(&protocol.ArgumentBuilder{
				// jsonAbi
				Type:        protocol.ARGUMENT_TYPE_STRING_VALUE,
				StringValue: jsonAbi,
			}).Build(),
			(&protocol.ArgumentBuilder{
				// ethTxHash
				Type:        protocol.ARGUMENT_TYPE_STRING_VALUE,
				StringValue: ethTxHash,
			}).Build(),
			(&protocol.ArgumentBuilder{
				// eventName
				Type:        protocol.ARGUMENT_TYPE_STRING_VALUE,
				StringValue: eventName,
			}).Build(),
		},
		PermissionScope: protocol.ExecutionPermissionScope(permissionScope),
	})
	if err != nil {
		panic(err.Error())
	}
	if len(output.OutputArguments) != 3 ||
		!output.OutputArguments[0].IsTypeBytesValue() ||
		!output.OutputArguments[1].IsTypeUint64Value() ||
		!output.OutputArguments[2].IsTypeUint32Value() {
		panic("getTransactionLog Sdk.Ethereum returned corrupt output value")
	}

	err = ethereum.ABIUnpackAllEventArguments(parsedABI, out, eventName, output.OutputArguments[0].BytesValue())
	if err != nil {
		panic(err.Error())
	}
	return output.OutputArguments[1].Uint64Value(), output.OutputArguments[2].Uint32Value()
}

func (s *service) SdkEthereumGetBlockNumber(executionContextId sdkContext.ContextId, permissionScope sdkContext.PermissionScope) (ethBlockNumber uint64) {
	output, err := s.sdkHandler.HandleSdkCall(context.TODO(), &handlers.HandleSdkCallInput{
		ContextId:       primitives.ExecutionContextId(executionContextId),
		OperationName:   SDK_OPERATION_NAME_ETHEREUM,
		MethodName:      "getBlockNumber",
		InputArguments:  []*protocol.Argument{},
		PermissionScope: protocol.ExecutionPermissionScope(permissionScope),
	})
	if err != nil {
		panic(err.Error())
	}
	if len(output.OutputArguments) != 1 || !output.OutputArguments[0].IsTypeUint64Value() {
		panic("getBlockNumber Sdk.Ethereum returned corrupt output value")
	}

	return output.OutputArguments[0].Uint64Value()
}

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

func (s *service) SdkEthereumCallMethod(executionContextId sdkContext.ContextId, permissionScope sdkContext.PermissionScope, contractAddress string, jsonAbi string, methodName string, out interface{}, args ...interface{}) {
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
				// contractAddress
				Type:        protocol.ARGUMENT_TYPE_STRING_VALUE,
				StringValue: contractAddress,
			}).Build(),
			(&protocol.ArgumentBuilder{
				// jsonAbi
				Type:        protocol.ARGUMENT_TYPE_STRING_VALUE,
				StringValue: jsonAbi,
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

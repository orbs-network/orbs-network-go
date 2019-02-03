package native

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
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
		s.logger.Panic("failed parsing ABI", log.Error(err), log.String("abi", jsonAbi))
	}

	packedInput, err := ethereum.ABIPackFunctionInputArguments(parsedABI, methodName, args)
	if err != nil {
		s.logger.Panic("failed packing input arguments", log.Error(err), log.String("method-name", methodName))
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
		s.logger.Panic("failed handling SDK call", log.Error(err))
	}
	if len(output.OutputArguments) != 1 || !output.OutputArguments[0].IsTypeBytesValue() {
		s.logger.Panic("callMethod Sdk.Ethereum returned corrupt output value", log.StringableSlice("output-arguments", output.OutputArguments))
	}

	err = ethereum.ABIUnpackFunctionOutputArguments(parsedABI, out, methodName, output.OutputArguments[0].BytesValue())
	if err != nil {
		s.logger.Panic("failed unpacking output arguments", log.Error(err), log.String("method-name", methodName), log.StringableSlice("output-arguments", output.OutputArguments))
	}
}

func (s *service) SdkEthereumGetTransactionLog(executionContextId sdkContext.ContextId, permissionScope sdkContext.PermissionScope, contractAddress string, jsonAbi string, ethTransactionId string, eventName string, out interface{}) {
	parsedABI, err := abi.JSON(strings.NewReader(jsonAbi))
	if err != nil {
		s.logger.Panic("failed parsing ABI", log.Error(err), log.String("abi", jsonAbi))
	}

	ethereumTxhash, err := hexutil.Decode(ethTransactionId)
	if err != nil {
		s.logger.Panic("failed decoding Ethereum tx id", log.Error(err), log.String("txid", ethTransactionId))
	}

	output, err := s.sdkHandler.HandleSdkCall(context.TODO(), &handlers.HandleSdkCallInput{
		ContextId:     primitives.ExecutionContextId(executionContextId),
		OperationName: SDK_OPERATION_NAME_ETHEREUM,
		MethodName:    "getTransactionLog",
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
				// ethereumTxhash
				Type:       protocol.ARGUMENT_TYPE_BYTES_VALUE,
				BytesValue: ethereumTxhash,
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
		s.logger.Panic("failed handling SDK call", log.Error(err))
	}
	if len(output.OutputArguments) != 1 || !output.OutputArguments[0].IsTypeBytesValue() {
		s.logger.Panic("getTransactionLog Sdk.Ethereum returned corrupt output value", log.StringableSlice("output-arguments", output.OutputArguments))
	}

	err = ethereum.ABIUnpackAllEventArguments(parsedABI, out, eventName, output.OutputArguments[0].BytesValue())
	if err != nil {
		s.logger.Panic("failed unpacking output arguments", log.Error(err), log.String("event-name", eventName), log.StringableSlice("output-arguments", output.OutputArguments))
	}
}

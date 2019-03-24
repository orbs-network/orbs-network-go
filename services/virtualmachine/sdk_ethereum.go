// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package virtualmachine

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

func (s *service) handleSdkEthereumCall(ctx context.Context, executionContext *executionContext, methodName primitives.MethodName, args []*protocol.Argument, permissionScope protocol.ExecutionPermissionScope) ([]*protocol.Argument, error) {
	switch methodName {

	case "callMethod":
		packedOutput, err := s.handleSdkEthereumCallMethod(ctx, executionContext, args, permissionScope)
		return []*protocol.Argument{(&protocol.ArgumentBuilder{
			// outputArgs
			Type:       protocol.ARGUMENT_TYPE_BYTES_VALUE,
			BytesValue: packedOutput,
		}).Build()}, err

	case "getTransactionLog":
		packedOutput, ethBlockNumber, ethTxIndex, err := s.handleSdkEthereumGetTransactionLog(ctx, executionContext, args, permissionScope)
		return []*protocol.Argument{(&protocol.ArgumentBuilder{
			// outputArgs
			Type:       protocol.ARGUMENT_TYPE_BYTES_VALUE,
			BytesValue: packedOutput,
		}).Build(), (&protocol.ArgumentBuilder{
			Type:        protocol.ARGUMENT_TYPE_UINT_64_VALUE,
			Uint64Value: ethBlockNumber,
		}).Build(), (&protocol.ArgumentBuilder{
			Type:        protocol.ARGUMENT_TYPE_UINT_32_VALUE,
			Uint32Value: ethTxIndex,
		}).Build()}, err

	case "getBlockNumber":
		ethBlockNumber, err := s.handleSdkEthereumGetBlockNumber(ctx, executionContext, args, permissionScope)
		return []*protocol.Argument{(&protocol.ArgumentBuilder{
			// outputArgs
			Type:        protocol.ARGUMENT_TYPE_UINT_64_VALUE,
			Uint64Value: ethBlockNumber,
		}).Build()}, err

	default:
		return nil, errors.Errorf("unknown SDK service call method: %s", methodName)
	}
}

// inputArg0: ethContractAddress (string)
// inputArg1: jsonAbi (string)
// inputArg2: ethBlockNumber (uint64)
// inputArg3: methodName (string)
// inputArg4: ethereumABIPackedInputArguments ([]byte)
// outputArg0: ethereumABIPackedOutput ([]byte)
func (s *service) handleSdkEthereumCallMethod(ctx context.Context, executionContext *executionContext, args []*protocol.Argument, permissionScope protocol.ExecutionPermissionScope) ([]byte, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	if len(args) != 5 || !args[0].IsTypeStringValue() || !args[1].IsTypeStringValue() || !args[2].IsTypeUint64Value() || !args[3].IsTypeStringValue() || !args[4].IsTypeBytesValue() {
		return nil, errors.Errorf("invalid SDK ethereum callMethod args: %v", args)
	}
	ethContractAddress := args[0].StringValue()
	jsonAbi := args[1].StringValue()
	ethBlockNumber := args[2].Uint64Value()
	methodName := args[3].StringValue()
	ethereumPackedInputArguments := args[4].BytesValue()

	// get block timestamp
	blockTimestamp := executionContext.currentBlockTimestamp

	// execute the call
	connector := s.crosschainConnectors[protocol.CROSSCHAIN_CONNECTOR_TYPE_ETHEREUM]
	output, err := connector.EthereumCallContract(ctx, &services.EthereumCallContractInput{
		ReferenceTimestamp:              blockTimestamp,
		EthereumBlockNumber:             ethBlockNumber,
		EthereumContractAddress:         ethContractAddress,
		EthereumFunctionName:            methodName,
		EthereumJsonAbi:                 jsonAbi,
		EthereumAbiPackedInputArguments: ethereumPackedInputArguments,
	})
	if err != nil {
		logger.Info("Sdk.Ethereum.CallMethod failed", log.Error(err), log.String("jsonAbi", jsonAbi))
		return nil, err
	}

	return output.EthereumAbiPackedOutput, nil
}

// inputArg0: ethContractAddress (string)
// inputArg1: jsonAbi (string)
// inputArg2: ethTxHash (string)
// inputArg3: eventName (string)
// outputArg0: ethereumABIPackedOutput ([]byte)
// outputArg1: ethBlockNumber (uint64)
// outputArg2: ethTxIndex (uint32)
func (s *service) handleSdkEthereumGetTransactionLog(ctx context.Context, executionContext *executionContext, args []*protocol.Argument, permissionScope protocol.ExecutionPermissionScope) ([]byte, uint64, uint32, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	if len(args) != 4 || !args[0].IsTypeStringValue() || !args[1].IsTypeStringValue() || !args[2].IsTypeStringValue() || !args[3].IsTypeStringValue() {
		return nil, 0, 0, errors.Errorf("invalid SDK ethereum getTransactionLog args: %v", args)
	}
	ethContractAddress := args[0].StringValue()
	jsonAbi := args[1].StringValue()
	ethTxHash := args[2].StringValue()
	eventName := args[3].StringValue()

	// get block timestamp
	blockTimestamp := executionContext.currentBlockTimestamp

	// execute the call
	connector := s.crosschainConnectors[protocol.CROSSCHAIN_CONNECTOR_TYPE_ETHEREUM]
	output, err := connector.EthereumGetTransactionLogs(ctx, &services.EthereumGetTransactionLogsInput{
		ReferenceTimestamp:      blockTimestamp,
		EthereumContractAddress: ethContractAddress,
		EthereumEventName:       eventName,
		EthereumJsonAbi:         jsonAbi,
		EthereumTxhash:          ethTxHash,
	})
	if err != nil {
		logger.Info("Sdk.Ethereum.GetTransactionLog failed", log.Error(err), log.String("jsonAbi", jsonAbi))
		return nil, 0, 0, err
	}
	if len(output.EthereumAbiPackedOutputs) == 0 {
		logger.Error("Sdk.Ethereum.GetTransactionLog returned zero results", log.String("jsonAbi", jsonAbi))
	}

	return output.EthereumAbiPackedOutputs[0], output.EthereumBlockNumber, output.EthereumTxindex, nil
}

// outputArg0: ethBlockNumber (uint64)
func (s *service) handleSdkEthereumGetBlockNumber(ctx context.Context, executionContext *executionContext, args []*protocol.Argument, permissionScope protocol.ExecutionPermissionScope) (uint64, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	if len(args) != 0 {
		return 0, errors.Errorf("invalid SDK ethereum getBlockNumber args: %v", args)
	}

	// get block timestamp
	blockTimestamp := executionContext.currentBlockTimestamp

	// execute the call
	connector := s.crosschainConnectors[protocol.CROSSCHAIN_CONNECTOR_TYPE_ETHEREUM]
	output, err := connector.EthereumGetBlockNumber(ctx, &services.EthereumGetBlockNumberInput{
		ReferenceTimestamp: blockTimestamp,
	})
	if err != nil {
		logger.Info("Sdk.Ethereum.GetBlockNumber failed", log.Error(err))
		return 0, err
	}

	return output.EthereumBlockNumber, nil
}

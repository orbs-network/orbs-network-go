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
		packedOutput, err := s.handleSdkEthereumGetTransactionLog(ctx, executionContext, args, permissionScope)
		return []*protocol.Argument{(&protocol.ArgumentBuilder{
			// outputArgs
			Type:       protocol.ARGUMENT_TYPE_BYTES_VALUE,
			BytesValue: packedOutput,
		}).Build()}, err

	default:
		return nil, errors.Errorf("unknown SDK service call method: %s", methodName)
	}
}

// inputArg0: contractAddress (string)
// inputArg1: jsonAbi (string)
// inputArg2: methodName (string)
// inputArg3: ethereumABIPackedInputArguments ([]byte)
// outputArg0: ethereumABIPackedOutput ([]byte)
func (s *service) handleSdkEthereumCallMethod(ctx context.Context, executionContext *executionContext, args []*protocol.Argument, permissionScope protocol.ExecutionPermissionScope) ([]byte, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	if len(args) != 4 || !args[0].IsTypeStringValue() || !args[1].IsTypeStringValue() || !args[2].IsTypeStringValue() || !args[3].IsTypeBytesValue() {
		return nil, errors.Errorf("invalid SDK ethereum callMethod args: %v", args)
	}
	contractAddress := args[0].StringValue()
	jsonAbi := args[1].StringValue()
	methodName := args[2].StringValue()
	ethereumPackedInputArguments := args[3].BytesValue()

	// get block timeatamp
	blockTimestamp := executionContext.blockTimestamp

	// execute the call
	connector := s.crosschainConnectors[protocol.CROSSCHAIN_CONNECTOR_TYPE_ETHEREUM]
	output, err := connector.EthereumCallContract(ctx, &services.EthereumCallContractInput{
		ReferenceTimestamp:              blockTimestamp,
		EthereumContractAddress:         contractAddress,
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

// inputArg0: contractAddress (string)
// inputArg1: jsonAbi (string)
// inputArg2: ethereumTxhash ([]byte)
// inputArg3: eventName (string)
// outputArg0: ethereumABIPackedOutput ([]byte)
func (s *service) handleSdkEthereumGetTransactionLog(ctx context.Context, executionContext *executionContext, args []*protocol.Argument, permissionScope protocol.ExecutionPermissionScope) ([]byte, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	if len(args) != 4 || !args[0].IsTypeStringValue() || !args[1].IsTypeStringValue() || !args[2].IsTypeBytesValue() || !args[3].IsTypeStringValue() {
		return nil, errors.Errorf("invalid SDK ethereum getTransactionLog args: %v", args)
	}
	contractAddress := args[0].StringValue()
	jsonAbi := args[1].StringValue()
	ethereumTxhash := args[2].BytesValue()
	eventName := args[3].StringValue()

	// execute the call
	connector := s.crosschainConnectors[protocol.CROSSCHAIN_CONNECTOR_TYPE_ETHEREUM]
	output, err := connector.EthereumGetTransactionLogs(ctx, &services.EthereumGetTransactionLogsInput{
		EthereumContractAddress: contractAddress,
		EthereumEventName:       eventName,
		EthereumJsonAbi:         jsonAbi,
		EthereumTxhash:          ethereumTxhash,
	})
	if err != nil {
		logger.Info("Sdk.Ethereum.GetTransactionLog failed", log.Error(err), log.String("jsonAbi", jsonAbi))
		return nil, err
	}

	return output.EthereumAbiPackedOutput, nil
}

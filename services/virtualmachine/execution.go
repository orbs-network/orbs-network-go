package virtualmachine

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

func (s *service) runMethod(
	blockHeight primitives.BlockHeight,
	transaction *protocol.Transaction,
	accessScope protocol.ExecutionAccessScope,
	batchTransientState *transientState,
) (protocol.ExecutionResult, []*protocol.MethodArgument, error) {

	// create execution context
	contextId, executionContext := s.contexts.allocateExecutionContext(blockHeight, accessScope)
	defer s.contexts.destroyExecutionContext(contextId)

	// get deployment info
	processor, contractPermission, err := s.getServiceDeployment(executionContext, transaction.ContractName())
	if err != nil {
		s.reporting.Info("get deployment for contract failed", log.Error(err), log.Stringable("transaction", transaction))
		return protocol.EXECUTION_RESULT_ERROR_UNEXPECTED, nil, err
	}

	// modify execution context
	executionContext.serviceStackPush(transaction.ContractName(), contractPermission)
	defer executionContext.serviceStackPop()
	executionContext.batchTransientState = batchTransientState

	// execute the call
	// TODO: might need to change protos to avoid this copy
	args := []*protocol.MethodArgument{}
	for i := transaction.InputArgumentsIterator(); i.HasNext(); {
		args = append(args, i.NextInputArguments())
	}
	output, err := processor.ProcessCall(&services.ProcessCallInput{
		ContextId:         contextId,
		ContractName:      transaction.ContractName(),
		MethodName:        transaction.MethodName(),
		InputArguments:    args,
		AccessScope:       accessScope,
		PermissionScope:   contractPermission,
		CallingService:    transaction.ContractName(),
		TransactionSigner: transaction.Signer(),
	})
	if err != nil {
		s.reporting.Info("transaction execution failed", log.Stringable("result", output.CallResult), log.Error(err), log.Stringable("transaction", transaction))
	}

	if batchTransientState != nil && output.CallResult == protocol.EXECUTION_RESULT_SUCCESS {
		executionContext.transientState.mergeIntoTransientState(batchTransientState)
	}

	return output.CallResult, output.OutputArguments, err
}

func (s *service) processTransactionSet(
	blockHeight primitives.BlockHeight,
	signedTransactions []*protocol.SignedTransaction,
) ([]*protocol.TransactionReceipt, []*protocol.ContractStateDiff) {

	// create batch transient state
	batchTransientState := newTransientState()

	// receipts for result
	receipts := make([]*protocol.TransactionReceipt, 0, len(signedTransactions))

	for _, signedTransaction := range signedTransactions {

		s.reporting.Info("processing transaction", log.Stringable("contract", signedTransaction.Transaction().ContractName()), log.Stringable("method", signedTransaction.Transaction().MethodName()), log.BlockHeight(blockHeight))
		callResult, outputArgs, _ := s.runMethod(blockHeight, signedTransaction.Transaction(), protocol.ACCESS_SCOPE_READ_WRITE, batchTransientState)

		receipt := s.encodeTransactionReceipt(signedTransaction.Transaction(), callResult, outputArgs)
		receipts = append(receipts, receipt)
	}

	stateDiffs := s.encodeBatchTransientStateToStateDiffs(batchTransientState)
	return receipts, stateDiffs
}

func (s *service) getRecentBlockHeight() (primitives.BlockHeight, primitives.TimestampNano, error) {
	output, err := s.stateStorage.GetStateStorageBlockHeight(&services.GetStateStorageBlockHeightInput{})
	if err != nil {
		return 0, 0, err
	}
	return output.LastCommittedBlockHeight, output.LastCommittedBlockTimestamp, nil
}

func (s *service) encodeTransactionReceipt(transaction *protocol.Transaction, result protocol.ExecutionResult, outputArgs []*protocol.MethodArgument) *protocol.TransactionReceipt {
	// TODO: might need to change protos to avoid this copy
	outputArgsBuilders := []*protocol.MethodArgumentBuilder{}
	for _, outputArg := range outputArgs {
		outputArgsBuilder := &protocol.MethodArgumentBuilder{
			Name: outputArg.Name(),
			Type: outputArg.Type(),
		}
		switch outputArg.Type() {
		case protocol.METHOD_ARGUMENT_TYPE_UINT_32_VALUE:
			outputArgsBuilder.Uint32Value = outputArg.Uint32Value()
		case protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE:
			outputArgsBuilder.Uint64Value = outputArg.Uint64Value()
		case protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE:
			outputArgsBuilder.StringValue = outputArg.StringValue()
		case protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE:
			outputArgsBuilder.BytesValue = outputArg.BytesValue()
		}
		outputArgsBuilders = append(outputArgsBuilders, outputArgsBuilder)
	}

	return (&protocol.TransactionReceiptBuilder{
		Txhash:          digest.CalcTxHash(transaction),
		ExecutionResult: result,
		OutputArguments: outputArgsBuilders,
	}).Build()
}

func (s *service) encodeBatchTransientStateToStateDiffs(batchTransientState *transientState) []*protocol.ContractStateDiff {
	res := []*protocol.ContractStateDiff{}
	for contractName, _ := range batchTransientState.contracts {
		stateDiffs := []*protocol.StateRecordBuilder{}
		batchTransientState.forDirty(contractName, func(key []byte, value []byte) {
			stateDiffs = append(stateDiffs, &protocol.StateRecordBuilder{
				Key:   key,
				Value: value,
			})
		})
		if len(stateDiffs) > 0 {
			res = append(res, (&protocol.ContractStateDiffBuilder{
				ContractName: contractName,
				StateDiffs:   stateDiffs,
			}).Build())
		}
	}
	return res
}

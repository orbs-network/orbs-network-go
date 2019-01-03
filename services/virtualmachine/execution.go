package virtualmachine

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

func (s *service) runMethod(
	ctx context.Context,
	lastCommittedBlockHeight primitives.BlockHeight,
	currentBlockHeight primitives.BlockHeight,
	currentBlockTimestamp primitives.TimestampNano,
	transaction *protocol.Transaction,
	accessScope protocol.ExecutionAccessScope,
	batchTransientState *transientState,
) (protocol.ExecutionResult, *protocol.ArgumentArray, *protocol.EventsArray, error) {

	// create execution context
	executionContextId, executionContext := s.contexts.allocateExecutionContext(lastCommittedBlockHeight, currentBlockHeight, currentBlockTimestamp, accessScope, transaction)
	defer s.contexts.destroyExecutionContext(executionContextId)
	executionContext.batchTransientState = batchTransientState

	// get deployment info
	processor, err := s.getServiceDeployment(ctx, executionContext, transaction.ContractName())
	if err != nil {
		s.logger.Info("get deployment info for contract failed", log.Error(err), log.Stringable("transaction", transaction))
		return protocol.EXECUTION_RESULT_ERROR_UNEXPECTED, nil, nil, err
	}

	// modify execution context
	executionContext.serviceStackPush(transaction.ContractName())
	defer executionContext.serviceStackPop()

	// execute the call
	inputArgs := protocol.ArgumentArrayReader(transaction.RawInputArgumentArrayWithHeader())
	output, err := processor.ProcessCall(ctx, &services.ProcessCallInput{
		ContextId:              executionContextId,
		ContractName:           transaction.ContractName(),
		MethodName:             transaction.MethodName(),
		InputArgumentArray:     inputArgs,
		AccessScope:            accessScope,
		CallingPermissionScope: protocol.PERMISSION_SCOPE_SERVICE,
		CallingService:         transaction.ContractName(),
	})
	if err != nil {
		s.logger.Info("transaction execution failed", log.Stringable("result", output.CallResult), log.Error(err), log.Stringable("transaction", transaction))
	}

	if batchTransientState != nil && output.CallResult == protocol.EXECUTION_RESULT_SUCCESS {
		executionContext.transientState.mergeIntoTransientState(batchTransientState)
	}

	outputEvents := (&protocol.EventsArrayBuilder{
		Events: executionContext.eventList,
	}).Build()

	return output.CallResult, output.OutputArgumentArray, outputEvents, err
}

func (s *service) processTransactionSet(
	ctx context.Context,
	currentBlockHeight primitives.BlockHeight,
	currentBlockTimestamp primitives.TimestampNano,
	signedTransactions []*protocol.SignedTransaction,
) ([]*protocol.TransactionReceipt, []*protocol.ContractStateDiff) {

	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	lastCommittedBlockHeight := currentBlockHeight - 1

	// create batch transient state
	batchTransientState := newTransientState()

	// receipts for result
	receipts := make([]*protocol.TransactionReceipt, 0, len(signedTransactions))

	for _, signedTransaction := range signedTransactions {

		logger.Info("processing transaction", log.Stringable("contract", signedTransaction.Transaction().ContractName()), log.Stringable("method", signedTransaction.Transaction().MethodName()), log.BlockHeight(currentBlockHeight))
		callResult, outputArgs, outputEvents, _ := s.runMethod(ctx, lastCommittedBlockHeight, currentBlockHeight, currentBlockTimestamp, signedTransaction.Transaction(), protocol.ACCESS_SCOPE_READ_WRITE, batchTransientState)
		if outputArgs == nil {
			outputArgs = (&protocol.ArgumentArrayBuilder{}).Build()
		}
		if outputEvents == nil {
			outputEvents = (&protocol.EventsArrayBuilder{}).Build()
		}

		receipt := s.encodeTransactionReceipt(signedTransaction.Transaction(), callResult, outputArgs, outputEvents)
		receipts = append(receipts, receipt)
	}

	stateDiffs := s.encodeBatchTransientStateToStateDiffs(batchTransientState)
	return receipts, stateDiffs
}

func (s *service) getRecentCommittedBlockHeight(ctx context.Context) (primitives.BlockHeight, primitives.TimestampNano, error) {
	output, err := s.stateStorage.GetStateStorageBlockHeight(ctx, &services.GetStateStorageBlockHeightInput{})
	if err != nil {
		return 0, 0, err
	}
	return output.LastCommittedBlockHeight, output.LastCommittedBlockTimestamp, nil
}

func (s *service) encodeTransactionReceipt(transaction *protocol.Transaction, result protocol.ExecutionResult, outputArgs *protocol.ArgumentArray, outputEvents *protocol.EventsArray) *protocol.TransactionReceipt {
	return (&protocol.TransactionReceiptBuilder{
		Txhash:              digest.CalcTxHash(transaction),
		ExecutionResult:     result,
		OutputEventsArray:   outputEvents.RawEventsArray(),
		OutputArgumentArray: outputArgs.RawArgumentsArray(),
	}).Build()
}

func (s *service) encodeBatchTransientStateToStateDiffs(batchTransientState *transientState) []*protocol.ContractStateDiff {
	res := []*protocol.ContractStateDiff{}
	for _, contractName := range batchTransientState.contractSortOrder {
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

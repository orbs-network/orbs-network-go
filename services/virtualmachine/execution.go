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
) (protocol.ExecutionResult, *protocol.MethodArgumentArray, error) {

	// create execution context
	contextId, executionContext := s.contexts.allocateExecutionContext(blockHeight, accessScope, transaction)
	defer s.contexts.destroyExecutionContext(contextId)

	// get deployment info
	processor, err := s.getServiceDeployment(executionContext, transaction.ContractName())
	if err != nil {
		s.logger.Info("get deployment info for contract failed", log.Error(err), log.Stringable("transaction", transaction))
		return protocol.EXECUTION_RESULT_ERROR_UNEXPECTED, nil, err
	}

	// modify execution context
	executionContext.serviceStackPush(transaction.ContractName())
	defer executionContext.serviceStackPop()
	executionContext.batchTransientState = batchTransientState

	// execute the call
	inputArgs := protocol.MethodArgumentArrayReader(transaction.RawInputArgumentArrayWithHeader())
	output, err := processor.ProcessCall(&services.ProcessCallInput{
		ContextId:              contextId,
		ContractName:           transaction.ContractName(),
		MethodName:             transaction.MethodName(),
		InputArgumentArray:     inputArgs,
		AccessScope:            accessScope,
		CallingPermissionScope: protocol.PERMISSION_SCOPE_SERVICE,
		CallingService:         transaction.ContractName(),
		TransactionSigner:      transaction.Signer(),
	})
	if err != nil {
		s.logger.Info("transaction execution failed", log.Stringable("result", output.CallResult), log.Error(err), log.Stringable("transaction", transaction))
	}

	if batchTransientState != nil && output.CallResult == protocol.EXECUTION_RESULT_SUCCESS {
		executionContext.transientState.mergeIntoTransientState(batchTransientState)
	}

	return output.CallResult, output.OutputArgumentArray, err
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

		s.logger.Info("processing transaction", log.Stringable("contract", signedTransaction.Transaction().ContractName()), log.Stringable("method", signedTransaction.Transaction().MethodName()), log.BlockHeight(blockHeight))
		callResult, outputArgs, _ := s.runMethod(blockHeight, signedTransaction.Transaction(), protocol.ACCESS_SCOPE_READ_WRITE, batchTransientState)
		if outputArgs == nil {
			outputArgs = (&protocol.MethodArgumentArrayBuilder{}).Build()
		}

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

func (s *service) encodeTransactionReceipt(transaction *protocol.Transaction, result protocol.ExecutionResult, outputArgs *protocol.MethodArgumentArray) *protocol.TransactionReceipt {
	return (&protocol.TransactionReceiptBuilder{
		Txhash:              digest.CalcTxHash(transaction),
		ExecutionResult:     result,
		OutputArgumentArray: outputArgs.RawArgumentsArray(),
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

package virtualmachine

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

func (s *service) processTransactionSet(
	blockHeight primitives.BlockHeight,
	signedTransactions []*protocol.SignedTransaction,
) ([]*protocol.TransactionReceipt, []*protocol.ContractStateDiff) {

	// create batch transient state
	batchTransientState := newTransientState()

	// receipts for result
	receipts := make([]*protocol.TransactionReceipt, 0, len(signedTransactions))

	for _, signedTransaction := range signedTransactions {

		// create execution context
		contextId, executionContext := s.contexts.allocateExecutionContext(blockHeight, protocol.ACCESS_SCOPE_READ_WRITE)
		defer s.contexts.destroyExecutionContext(contextId)
		executionContext.serviceStackPush(signedTransaction.Transaction().ContractName())
		executionContext.batchTransientState = batchTransientState

		// execute the call
		// TODO: might need to change protos to avoid this copy
		args := []*protocol.MethodArgument{}
		for i := signedTransaction.Transaction().InputArgumentsIterator(); i.HasNext(); {
			args = append(args, i.NextInputArguments())
		}
		output, err := s.processors[protocol.PROCESSOR_TYPE_NATIVE].ProcessCall(&services.ProcessCallInput{
			ContextId:         contextId,
			ContractName:      signedTransaction.Transaction().ContractName(),
			MethodName:        signedTransaction.Transaction().MethodName(),
			InputArguments:    args,
			AccessScope:       protocol.ACCESS_SCOPE_READ_WRITE,
			PermissionScope:   protocol.PERMISSION_SCOPE_SERVICE, // TODO: improve
			CallingService:    signedTransaction.Transaction().ContractName(),
			TransactionSigner: signedTransaction.Transaction().Signer(),
		})
		if err != nil {
			s.reporting.Info("processTransactionSet process transaction failed", instrumentation.Error(err), instrumentation.Stringable("transaction", signedTransaction.Transaction()))
		}

		receipt := s.encodeTransactionReceipt(signedTransaction.Transaction(), output.CallResult, output.OutputArguments)
		receipts = append(receipts, receipt)

		if output.CallResult == protocol.EXECUTION_RESULT_SUCCESS {
			executionContext.transientState.mergeIntoTransientState(batchTransientState)
		}

	}

	stateDiffs := s.encodeBatchTransientStateToStateDiffs(batchTransientState)
	return receipts, stateDiffs
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

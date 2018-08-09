package virtualmachine

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

func (s *service) processTransactionSet(
	blockHeight primitives.BlockHeight,
	transactions []*protocol.SignedTransaction,
) ([]*protocol.ContractStateDiff, error) {

	// create batch transient state
	batchTransientState := newTransientState()

	for _, signedTransaction := range transactions {

		// create execution context
		contextId, executionContext := s.contexts.allocateExecutionContext(blockHeight, protocol.ACCESS_SCOPE_READ_WRITE)
		defer s.contexts.destroyExecutionContext(contextId)
		executionContext.serviceStackPush(signedTransaction.Transaction().ContractName())
		executionContext.batchTransientState = batchTransientState

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
			return nil, err
		}

		if output.CallResult == protocol.EXECUTION_RESULT_SUCCESS {
			executionContext.transientState.mergeIntoTransientState(batchTransientState)
		}

	}

	stateDiffs := s.encodeBatchTransientStateToStateDiffs(batchTransientState)
	return stateDiffs, nil
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

package virtualmachine

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_GlobalPreOrder"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

func (s *service) callGlobalPreOrderSystemContract(ctx context.Context, blockHeight primitives.BlockHeight, blockTimestamp primitives.TimestampNano) error {
	systemContractName := primitives.ContractName(globalpreorder_systemcontract.CONTRACT_NAME)
	systemMethodName := primitives.MethodName(globalpreorder_systemcontract.METHOD_APPROVE)

	// create execution context
	executionContextId, executionContext := s.contexts.allocateExecutionContext(blockHeight, blockTimestamp, protocol.ACCESS_SCOPE_READ_ONLY, nil)
	defer s.contexts.destroyExecutionContext(executionContextId)

	// modify execution context
	executionContext.serviceStackPush(systemContractName)
	defer executionContext.serviceStackPop()

	// execute the call
	_, err := s.processors[protocol.PROCESSOR_TYPE_NATIVE].ProcessCall(ctx, &services.ProcessCallInput{
		ContextId:              executionContextId,
		ContractName:           systemContractName,
		MethodName:             systemMethodName,
		InputArgumentArray:     (&protocol.ArgumentArrayBuilder{}).Build(),
		AccessScope:            protocol.ACCESS_SCOPE_READ_ONLY,
		CallingPermissionScope: protocol.PERMISSION_SCOPE_SERVICE,
		CallingService:         systemContractName,
	})

	return err
}

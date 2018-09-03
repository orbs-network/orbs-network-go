package virtualmachine

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_GlobalPreOrder"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

func (s *service) callGlobalPreOrderSystemContract(blockHeight primitives.BlockHeight) error {
	systemContractName := globalpreorder.CONTRACT.Name
	systemMethodName := globalpreorder.METHOD_APPROVE.Name

	// create execution context
	contextId, executionContext := s.contexts.allocateExecutionContext(blockHeight, protocol.ACCESS_SCOPE_READ_ONLY)
	defer s.contexts.destroyExecutionContext(contextId)

	// modify execution context
	executionContext.serviceStackPush(systemContractName)
	defer executionContext.serviceStackPop()

	// execute the call
	_, err := s.processors[protocol.PROCESSOR_TYPE_NATIVE].ProcessCall(&services.ProcessCallInput{
		ContextId:              contextId,
		ContractName:           systemContractName,
		MethodName:             systemMethodName,
		InputArgumentArray:     (&protocol.MethodArgumentArrayBuilder{}).Build(),
		AccessScope:            protocol.ACCESS_SCOPE_READ_ONLY,
		CallingPermissionScope: protocol.PERMISSION_SCOPE_SERVICE,
		CallingService:         systemContractName,
		TransactionSigner:      nil,
	})

	return err
}

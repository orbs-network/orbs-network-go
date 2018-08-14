package virtualmachine

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_GlobalPreOrder"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

func (s *service) callGlobalPreOrderContract(blockHeight primitives.BlockHeight) error {
	contractName := globalpreorder.CONTRACT.Name
	methodName := globalpreorder.METHOD_APPROVE.Name
	contractPermissions := globalpreorder.CONTRACT.Permission

	// create execution context
	contextId, executionContext := s.contexts.allocateExecutionContext(blockHeight, protocol.ACCESS_SCOPE_READ_ONLY)
	defer s.contexts.destroyExecutionContext(contextId)
	executionContext.serviceStackPush(contractName)

	// execute the call
	_, err := s.processors[protocol.PROCESSOR_TYPE_NATIVE].ProcessCall(&services.ProcessCallInput{
		ContextId:         contextId,
		ContractName:      contractName,
		MethodName:        methodName,
		InputArguments:    []*protocol.MethodArgument{},
		AccessScope:       protocol.ACCESS_SCOPE_READ_ONLY,
		PermissionScope:   contractPermissions, // TODO: kill this argument https://github.com/orbs-network/orbs-spec/issues/64
		CallingService:    contractName,
		TransactionSigner: nil,
	})

	return err
}

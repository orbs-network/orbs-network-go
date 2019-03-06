package virtualmachine

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

func (s *service) callSystemContract(
	ctx context.Context,
	blockHeight primitives.BlockHeight,
	blockTimestamp primitives.TimestampNano,
	systemContractName primitives.ContractName,
	systemMethodName primitives.MethodName,
	inputArgs *protocol.ArgumentArray,
) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {

	// create execution context
	executionContextId, executionContext := s.contexts.allocateExecutionContext(blockHeight, blockHeight, blockTimestamp, protocol.ACCESS_SCOPE_READ_ONLY, nil)
	defer s.contexts.destroyExecutionContext(executionContextId)

	// modify execution context
	executionContext.serviceStackPush(systemContractName)
	defer executionContext.serviceStackPop()

	// execute the call
	output, err := s.processors[protocol.PROCESSOR_TYPE_NATIVE].ProcessCall(ctx, &services.ProcessCallInput{
		ContextId:              executionContextId,
		ContractName:           systemContractName,
		MethodName:             systemMethodName,
		InputArgumentArray:     inputArgs,
		AccessScope:            protocol.ACCESS_SCOPE_READ_ONLY,
		CallingPermissionScope: protocol.PERMISSION_SCOPE_SERVICE,
	})

	return output.CallResult, output.OutputArgumentArray, err
}

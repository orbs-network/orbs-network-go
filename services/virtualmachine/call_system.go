// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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

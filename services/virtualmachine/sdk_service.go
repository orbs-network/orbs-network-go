// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package virtualmachine

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

func (s *service) handleSdkServiceCall(ctx context.Context, executionContext *executionContext, methodName primitives.MethodName, args []*protocol.Argument, permissionScope protocol.ExecutionPermissionScope) ([]*protocol.Argument, error) {
	switch methodName {

	case "callMethod":
		outputArgumentArrayRaw, err := s.handleSdkServiceCallMethod(ctx, executionContext, args, permissionScope)
		return []*protocol.Argument{(&protocol.ArgumentBuilder{
			// outputArgs
			Type:       protocol.ARGUMENT_TYPE_BYTES_VALUE,
			BytesValue: outputArgumentArrayRaw,
		}).Build()}, err

	default:
		return nil, errors.Errorf("unknown SDK service call method: %s", methodName)
	}
}

// inputArg0: serviceName (string)
// inputArg1: methodName (string)
// inputArg2: inputArgumentArray ([]byte of raw ArgumentArray)
// outputArg0: outputArgumentArray ([]byte of raw ArgumentArray)
func (s *service) handleSdkServiceCallMethod(ctx context.Context, executionContext *executionContext, args []*protocol.Argument, permissionScope protocol.ExecutionPermissionScope) ([]byte, error) {
	if len(args) != 3 || !args[0].IsTypeStringValue() || !args[1].IsTypeStringValue() || !args[2].IsTypeBytesValue() {
		return nil, errors.Errorf("invalid SDK service callMethod args: %v", args)
	}
	serviceName := args[0].StringValue()
	methodName := args[1].StringValue()
	inputArgumentArray := protocol.ArgumentArrayReader(args[2].BytesValue())

	// get deployment info
	processor, err := s.getServiceDeployment(ctx, executionContext, primitives.ContractName(serviceName))
	if err != nil {
		s.logger.Info("get deployment info for contract failed during Sdk.Service.CallMethod", log.Error(err), log.String("contract", serviceName))
		return nil, err
	}

	// modify execution context
	executionContext.serviceStackPush(primitives.ContractName(serviceName))
	defer executionContext.serviceStackPop()

	// execute the call
	output, err := processor.ProcessCall(ctx, &services.ProcessCallInput{
		ContextId:              executionContext.contextId,
		ContractName:           primitives.ContractName(serviceName),
		MethodName:             primitives.MethodName(methodName),
		InputArgumentArray:     inputArgumentArray,
		AccessScope:            executionContext.accessScope,
		CallingPermissionScope: permissionScope,
	})
	if err != nil {
		s.logger.Info("Sdk.Service.CallMethod failed", log.Error(err), log.Stringable("callee", primitives.ContractName(serviceName)))
		return nil, err
	}

	return output.OutputArgumentArray.Raw(), nil
}

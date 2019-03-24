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
	"github.com/pkg/errors"
)

func (s *service) handleSdkEventsCall(ctx context.Context, executionContext *executionContext, methodName primitives.MethodName, args []*protocol.Argument, permissionScope protocol.ExecutionPermissionScope) ([]*protocol.Argument, error) {
	switch methodName {

	case "emitEvent":
		err := s.handleSdkEventsEmitEvent(ctx, executionContext, args, permissionScope)
		if err != nil {
			return nil, err
		}
		return []*protocol.Argument{}, nil

	default:
		return nil, errors.Errorf("unknown SDK events call method: %s", methodName)
	}
}

// inputArg0: eventName (string)
// inputArg1: inputArgumentArray ([]byte of raw ArgumentArray)
func (s *service) handleSdkEventsEmitEvent(ctx context.Context, executionContext *executionContext, args []*protocol.Argument, permissionScope protocol.ExecutionPermissionScope) error {
	if len(args) != 2 || !args[0].IsTypeStringValue() || !args[1].IsTypeBytesValue() {
		return errors.Errorf("invalid SDK events callMethod args: %v", args)
	}
	eventName := args[0].StringValue()
	inputArgumentArray := protocol.ArgumentArrayReader(args[1].BytesValue())

	executionContext.eventListAdd(primitives.EventName(eventName), inputArgumentArray.RawArgumentsArray())

	return nil
}

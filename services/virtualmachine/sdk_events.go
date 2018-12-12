package virtualmachine

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

func (s *service) handleSdkEventsCall(ctx context.Context, executionContext *executionContext, methodName primitives.MethodName, args []*protocol.MethodArgument, permissionScope protocol.ExecutionPermissionScope) ([]*protocol.MethodArgument, error) {
	switch methodName {

	case "emitEvent":
		err := s.handleSdkEventsEmitEvent(ctx, executionContext, args, permissionScope)
		if err != nil {
			return nil, err
		}
		return []*protocol.MethodArgument{}, nil

	default:
		return nil, errors.Errorf("unknown SDK events call method: %s", methodName)
	}
}

// inputArg0: eventName (string)
// inputArg1: inputArgumentArray ([]byte of raw MethodArgumentArray)
func (s *service) handleSdkEventsEmitEvent(ctx context.Context, executionContext *executionContext, args []*protocol.MethodArgument, permissionScope protocol.ExecutionPermissionScope) error {
	if len(args) != 2 || !args[0].IsTypeStringValue() || !args[1].IsTypeBytesValue() {
		return errors.Errorf("invalid SDK events callMethod args: %v", args)
	}
	eventName := args[0].StringValue()
	inputArgumentArray := protocol.MethodArgumentArrayReader(args[1].BytesValue())

	executionContext.eventListAdd(primitives.EventName(eventName), inputArgumentArray.RawArgumentsArray())

	return nil
}

package virtualmachine

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

func (s *service) handleSdkEnvCall(ctx context.Context, executionContext *executionContext, methodName primitives.MethodName, args []*protocol.Argument, permissionScope protocol.ExecutionPermissionScope) ([]*protocol.Argument, error) {
	switch methodName {

	case "getBlockHeight":
		value, err := s.handleSdkEnvGetBlockHeight(executionContext, args)
		if err != nil {
			return nil, err
		}
		return []*protocol.Argument{(&protocol.ArgumentBuilder{
			// value
			Type:        protocol.ARGUMENT_TYPE_UINT_64_VALUE,
			Uint64Value: value,
		}).Build()}, nil

	case "getBlockTimestamp":
		value, err := s.handleSdkEnvGetBlockTimestamp(executionContext, args)
		if err != nil {
			return nil, err
		}
		return []*protocol.Argument{(&protocol.ArgumentBuilder{
			// value
			Type:        protocol.ARGUMENT_TYPE_UINT_64_VALUE,
			Uint64Value: value,
		}).Build()}, nil

	default:
		return nil, errors.Errorf("unknown SDK env call method: %s", methodName)
	}
}

// outputArg0: value (uint64)
func (s *service) handleSdkEnvGetBlockHeight(executionContext *executionContext, args []*protocol.Argument) (uint64, error) {
	if len(args) != 0 {
		return 0, errors.Errorf("invalid SDK env getBlockHeight args: %v", args)
	}

	return uint64(executionContext.blockHeight), nil
}

// outputArg0: value (uint64)
func (s *service) handleSdkEnvGetBlockTimestamp(executionContext *executionContext, args []*protocol.Argument) (uint64, error) {
	if len(args) != 0 {
		return 0, errors.Errorf("invalid SDK env getBlockTimestamp args: %v", args)
	}

	return uint64(executionContext.blockTimestamp), nil
}

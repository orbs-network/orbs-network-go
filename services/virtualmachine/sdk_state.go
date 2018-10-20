package virtualmachine

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

func (s *service) handleSdkStateCall(ctx context.Context, executionContext *executionContext, methodName primitives.MethodName, args []*protocol.MethodArgument, permissionScope protocol.ExecutionPermissionScope) ([]*protocol.MethodArgument, error) {
	switch methodName {

	case "read":
		value, err := s.handleSdkStateRead(ctx, executionContext, args)
		if err != nil {
			return nil, err
		}
		return []*protocol.MethodArgument{(&protocol.MethodArgumentBuilder{
			Name:       "value",
			Type:       protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE,
			BytesValue: value,
		}).Build()}, nil

	case "write":
		err := s.handleSdkStateWrite(executionContext, args)
		if err != nil {
			return nil, err
		}
		return []*protocol.MethodArgument{}, nil

	default:
		return nil, errors.Errorf("unknown SDK state call method: %s", methodName)
	}
}

// inputArg0: key ([]byte)
// outputArg0: value ([]byte)
func (s *service) handleSdkStateRead(ctx context.Context, executionContext *executionContext, args []*protocol.MethodArgument) ([]byte, error) {
	if len(args) != 1 || !args[0].IsTypeBytesValue() {
		return nil, errors.Errorf("invalid SDK state read args: %v", args)
	}
	key := args[0].BytesValue()

	// get current running service
	currentService := executionContext.serviceStackTop()

	// try from transient state first
	value, found := executionContext.transientState.getValue(currentService, key)
	if found {
		return value, nil
	}

	// try from batch transient state first
	if executionContext.batchTransientState != nil {
		value, found = executionContext.batchTransientState.getValue(currentService, key)
		if found {
			return value, nil
		}
	}

	// cache miss to state storage
	output, err := s.stateStorage.ReadKeys(ctx, &services.ReadKeysInput{
		BlockHeight:  executionContext.blockHeight,
		ContractName: currentService,
		Keys:         []primitives.Ripmd160Sha256{key},
	})
	if err != nil {
		return nil, err
	}
	if len(output.StateRecords) == 0 {
		return nil, errors.Errorf("state read returned no value")
	}
	value = output.StateRecords[0].Value()

	// store in transient state (cache)
	executionContext.transientState.setValue(currentService, key, value, false)

	return value, nil
}

// inputArg0: key ([]byte)
// inputArg1: value ([]byte)
func (s *service) handleSdkStateWrite(executionContext *executionContext, args []*protocol.MethodArgument) error {
	if executionContext.accessScope != protocol.ACCESS_SCOPE_READ_WRITE {
		return errors.Errorf("write attempted without write access: %s", executionContext.accessScope)
	}

	if len(args) != 2 || !args[0].IsTypeBytesValue() || !args[1].IsTypeBytesValue() {
		return errors.Errorf("invalid SDK state write args: %v", args)
	}
	key := args[0].BytesValue()
	value := args[1].BytesValue()

	// get current running service
	currentService := executionContext.serviceStackTop()

	// write to transient state
	// TODO: maybe compare with getValue to see the value actually changed
	executionContext.transientState.setValue(currentService, key, value, true)

	return nil
}

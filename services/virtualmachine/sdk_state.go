package virtualmachine

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

func (s *service) handleSdkStateCall(context *executionContext, methodName primitives.MethodName, args []*protocol.MethodArgument) ([]*protocol.MethodArgument, error) {
	switch methodName {
	case "read":
		return s.handleSdkStateRead(context, args)
	case "write":
		return s.handleSdkStateWrite(context, args)
	default:
		return nil, errors.Errorf("unknown SDK state call method: %s", methodName)
	}
}

func (s *service) handleSdkStateRead(context *executionContext, args []*protocol.MethodArgument) ([]*protocol.MethodArgument, error) {
	if len(args) == 0 || !args[0].IsTypeBytesValue() {
		return nil, errors.Errorf("invalid SDK state read args: %v", args)
	}
	output, err := s.stateStorage.ReadKeys(&services.ReadKeysInput{
		BlockHeight:  context.blockHeight,
		ContractName: context.serviceStackTop(),
		Keys:         []primitives.Ripmd160Sha256{args[0].BytesValue()},
	})
	if err != nil {
		return nil, err
	}
	if len(output.StateRecords) == 0 {
		return nil, errors.Errorf("state read returned no value")
	}
	res := []*protocol.MethodArgument{(&protocol.MethodArgumentBuilder{
		Name:       "value",
		Type:       protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE,
		BytesValue: output.StateRecords[0].Value(),
	}).Build()}
	return res, nil
}

func (s *service) handleSdkStateWrite(context *executionContext, args []*protocol.MethodArgument) ([]*protocol.MethodArgument, error) {
	if context.transientState == nil {
		return nil, errors.Errorf("write attempted without transient state: %v", args)
	}
	return nil, nil
}

package virtualmachine

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

func (s *service) handleSdkAddressCall(context *executionContext, methodName primitives.MethodName, args []*protocol.MethodArgument, permissionScope protocol.ExecutionPermissionScope) ([]*protocol.MethodArgument, error) {
	switch methodName {

	case "getSignerAddress":
		value, err := s.handleSdkAddressGetSignerAddress(context, args)
		if err != nil {
			return nil, err
		}
		return []*protocol.MethodArgument{(&protocol.MethodArgumentBuilder{
			Name:       "value",
			Type:       protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE,
			BytesValue: value,
		}).Build()}, nil

	case "getCallerAddress":
		value, err := s.handleSdkAddressGetCallerAddress(context, args)
		if err != nil {
			return nil, err
		}
		return []*protocol.MethodArgument{(&protocol.MethodArgumentBuilder{
			Name:       "value",
			Type:       protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE,
			BytesValue: value,
		}).Build()}, nil

	default:
		return nil, errors.Errorf("unknown SDK address call method: %s", methodName)
	}
}

// outputArg0: value ([]byte)
func (s *service) handleSdkAddressGetSignerAddress(context *executionContext, args []*protocol.MethodArgument) ([]byte, error) {
	if len(args) != 0 {
		return nil, errors.Errorf("invalid SDK address getSignerAddress args: %v", args)
	}

	if context.transaction == nil {
		return nil, errors.New("operation does not contain a transaction")
	}

	return s.getSignerAddress(context.transaction.Signer())
}

// outputArg0: value ([]byte)
func (s *service) handleSdkAddressGetCallerAddress(context *executionContext, args []*protocol.MethodArgument) ([]byte, error) {
	if len(args) != 0 {
		return nil, errors.Errorf("invalid SDK address getCallerAddress args: %v", args)
	}

	if context.serviceStackDepth() == 1 {
		// on the first caller, fallback to GetSignerAddress
		return s.handleSdkAddressGetSignerAddress(context, args)
	} else {
		// after a contract call, get the caller address
		return addressContractCall(context.serviceStackPeekCaller())
	}
}

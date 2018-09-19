package virtualmachine

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

func (s *service) handleSdkAddressCall(context *executionContext, methodName primitives.MethodName, args []*protocol.MethodArgument, permissionScope protocol.ExecutionPermissionScope) ([]*protocol.MethodArgument, error) {
	switch methodName {

	case "getSignerAddress":
		return nil, nil

	case "getCallerAddress":
		return nil, nil

	default:
		return nil, errors.Errorf("unknown SDK address call method: %s", methodName)
	}
}

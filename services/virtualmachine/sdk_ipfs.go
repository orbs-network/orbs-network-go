package virtualmachine

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/ipfs"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

func (s *service) handleSdkIPFSCall(ctx context.Context, executionContext *executionContext, methodName primitives.MethodName, args []*protocol.Argument, permissionScope protocol.ExecutionPermissionScope) ([]*protocol.Argument, error) {
	switch methodName {

	case "read":
		value, err := s.ipfs.Read(ctx, &ipfs.IPFSReadInput{
			Hash: args[0].StringValue(),
		})
		if err != nil {
			return nil, err
		}

		return []*protocol.Argument{(&protocol.ArgumentBuilder{
			// value
			Type:       protocol.ARGUMENT_TYPE_BYTES_VALUE,
			BytesValue: value.Content,
		}).Build()}, nil
	default:
		return nil, errors.Errorf("unknown SDK env call method: %s", methodName)
	}
}

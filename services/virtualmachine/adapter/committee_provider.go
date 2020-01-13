package adapter

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type CommitteeProvider interface {
	GetCommittee(ctx context.Context, referenceNumber uint64) ([]primitives.NodeAddress, error)
}

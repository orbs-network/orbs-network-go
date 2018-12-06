package adapter

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type BlockHeader struct {
	Number    int64
	Timestamp int64
}

func (c *connectorCommon) GetBlockByTimestamp(ctx context.Context, nano primitives.TimestampNano) (*BlockHeader, error) {
	return &BlockHeader{0, 0}, nil
}

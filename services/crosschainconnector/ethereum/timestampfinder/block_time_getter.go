package timestampfinder

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"math/big"
)

type BlockNumberAndTime struct {
	BlockNumber   int64
	BlockTimeNano primitives.TimestampNano
}

type BlockTimeGetter interface {
	GetTimestampForBlockNumber(ctx context.Context, blockNumber *big.Int) (*BlockNumberAndTime, error)
	GetTimestampForLatestBlock(ctx context.Context) (*BlockNumberAndTime, error)
}

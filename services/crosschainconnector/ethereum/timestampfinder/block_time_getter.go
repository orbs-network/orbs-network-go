package timestampfinder

import (
	"context"
	"math/big"
)

type BlockNumberAndTime struct {
	BlockNumber int64
	TimeSeconds int64
}

type BlockTimeGetter interface {
	GetTimestampForBlockNumber(ctx context.Context, blockNumber *big.Int) (*BlockNumberAndTime, error)
}

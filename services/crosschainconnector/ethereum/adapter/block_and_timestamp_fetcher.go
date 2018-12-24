package adapter

import (
	"context"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
)

type BlockAndTimestampGetter interface {
	ApproximateBlockAt(ctx context.Context, blockNumber *big.Int) (*BlockHeightAndTime, error)
}

type blockHeaderFetcher interface {
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
}

type EthereumBasedBlockAndTimestampGetter struct {
	ethereum blockHeaderFetcher
}

func NewBlockTimestampFetcher(ethereum blockHeaderFetcher) *EthereumBasedBlockAndTimestampGetter {
	return &EthereumBasedBlockAndTimestampGetter{ethereum}
}

func (f *EthereumBasedBlockAndTimestampGetter) ApproximateBlockAt(ctx context.Context, blockNumber *big.Int) (*BlockHeightAndTime, error) {
	header, err := f.ethereum.HeaderByNumber(ctx, blockNumber)
	if err != nil {
		return nil, err
	}

	if header == nil { // simulator always returns nil block number
		return nil, nil
	}

	return &BlockHeightAndTime{Time: header.Time.Int64(), Number: header.Number.Int64()}, nil
}

type BlockHeightAndTime struct {
	Number int64
	Time   int64
}

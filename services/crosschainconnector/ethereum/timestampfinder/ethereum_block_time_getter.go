package timestampfinder

import (
	"context"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
)

type adapterHeaderFetcher interface {
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
}

type EthereumBasedBlockTimeGetter struct {
	ethereum adapterHeaderFetcher
}

func NewEthereumBasedBlockTimeGetter(ethereum adapterHeaderFetcher) *EthereumBasedBlockTimeGetter {
	return &EthereumBasedBlockTimeGetter{ethereum}
}

func (f *EthereumBasedBlockTimeGetter) GetTimestampForBlockNumber(ctx context.Context, blockNumber *big.Int) (*BlockNumberAndTime, error) {
	header, err := f.ethereum.HeaderByNumber(ctx, blockNumber)
	if err != nil {
		return nil, err
	}

	if header == nil { // simulator always returns nil block number
		return nil, nil
	}

	return &BlockNumberAndTime{TimeSeconds: header.Time.Int64(), BlockNumber: header.Number.Int64()}, nil
}

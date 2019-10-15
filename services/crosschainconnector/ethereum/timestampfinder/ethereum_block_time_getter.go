// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package timestampfinder

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"math/big"
)

type adapterHeaderFetcher interface {
	HeaderByNumber(ctx context.Context, number *big.Int) (*adapter.BlockNumberAndTime, error)
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

	// TODO	https://github.com/orbs-network/orbs-network-go/issues/1214 simulator always returns nil block number
	if header == nil {
		return nil, nil
	}

	return &BlockNumberAndTime{
		BlockNumber:   header.BlockNumber,
		BlockTimeNano: secondsToNano(header.TimeInSeconds),
	}, nil
}

func (f *EthereumBasedBlockTimeGetter) GetTimestampForLatestBlock(ctx context.Context) (*BlockNumberAndTime, error) {
	// ethereum regards nil block number as latest
	return f.GetTimestampForBlockNumber(ctx, nil)
}

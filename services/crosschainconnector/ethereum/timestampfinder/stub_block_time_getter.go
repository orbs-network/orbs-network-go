// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package timestampfinder

import (
	"context"
	"github.com/pkg/errors"
	"math/big"
)

type getterStub struct {
	blocks []BlockNumberAndTime
}

func newBlockTimeGetterStub(blocks []BlockNumberAndTime) BlockTimeGetter {
	return &getterStub{blocks: blocks}
}

func (g *getterStub) GetTimestampForBlockNumber(ctx context.Context, blockNumber *big.Int) (*BlockNumberAndTime, error) {
	for _, block := range g.blocks {
		if block.BlockNumber == blockNumber.Int64() {
			return &block, nil
		}
	}
	return nil, errors.Errorf("could not find block number %v", blockNumber)
}

func (g *getterStub) GetTimestampForLatestBlock(ctx context.Context) (*BlockNumberAndTime, error) {
	if len(g.blocks) == 0 {
		return nil, errors.New("no blocks found")
	}
	return &g.blocks[len(g.blocks)-1], nil
}

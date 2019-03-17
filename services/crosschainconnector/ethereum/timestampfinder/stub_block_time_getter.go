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

package internalsync

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type blockSyncFunc func (ctx context.Context, committedBlockPair *protocol.BlockPairContainer) (primitives.BlockHeight, error)

type blockSource interface {
	GetBlockTracker() *synchronization.BlockTracker
	GetBlocks(first primitives.BlockHeight, last primitives.BlockHeight) (blocks []*protocol.BlockPairContainer, firstReturnedBlockHeight primitives.BlockHeight, lastReturnedBlockHeight primitives.BlockHeight, err error)
	GetNumBlocks() (primitives.BlockHeight, error)
}

func syncOnce(ctx context.Context, source blockSource, callback blockSyncFunc) (primitives.BlockHeight, error) {
	sourceTopHeight, err := source.GetNumBlocks()
	if err != nil {
		return 0, err
	}

	for i := sourceTopHeight; i <= sourceTopHeight; {
		singleBlockArr, _, _, err := source.GetBlocks(i, i)
		if err != nil {
			return 0, err
		}
		nextHeight, err := callback(ctx, singleBlockArr[0])
		if err != nil {
			return 0, err
		}
		if nextHeight == i {
			return 0, fmt.Errorf("failed to sync block at height %d", i)
		}
		i = nextHeight
	}

	return sourceTopHeight, nil
}

func StartSupervised(ctx context.Context, logger supervised.Errorer, source blockSource, callback blockSyncFunc) {
	supervised.GoForever(ctx, logger, func() {
		height, err := syncOnce(ctx, source, callback)
		for err == nil {
			err = source.GetBlockTracker().WaitForBlock(ctx, height + 1)
			if err != nil {
				return
			}
			height, err = syncOnce(ctx, source, callback)
		}
	})
}
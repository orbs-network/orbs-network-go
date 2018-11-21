package servicesync

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type BlockPairCommitter interface {
	commitBlockPair(ctx context.Context, committedBlockPair *protocol.BlockPairContainer) (primitives.BlockHeight, error)
	getServiceName() string
}

type blockSource interface {
	GetBlockTracker() *synchronization.BlockTracker
	GetBlocks(first primitives.BlockHeight, last primitives.BlockHeight) (blocks []*protocol.BlockPairContainer, firstReturnedBlockHeight primitives.BlockHeight, lastReturnedBlockHeight primitives.BlockHeight, err error)
	GetLastBlock() (*protocol.BlockPairContainer, error)
}

func syncOnce(ctx context.Context, source blockSource, committer BlockPairCommitter, logger log.BasicLogger) (primitives.BlockHeight, error) {
	topBlock, err := source.GetLastBlock()
	if err != nil {
		return 0, err
	}
	topBlockHeight := topBlock.ResultsBlock.Header.BlockHeight()

	for i := topBlockHeight; i <= topBlockHeight; {
		singleBlockArr, _, _, err := source.GetBlocks(i, i+1) // GetBlocks is 1 based for some strange reason
		if err != nil {
			return 0, err
		}
		bp := singleBlockArr[0]

		// Log each transaction being synced TODO - move this from here into the callback func or just relax logging / write under debug level when available
		h := bp.ResultsBlock.Header.BlockHeight()
		for _, tx := range bp.ResultsBlock.TransactionReceipts {
			logger.Info("attempt service sync for block", log.BlockHeight(h), log.Transaction(tx.Txhash()))
		}

		// notify the receiving service of the new block
		nextHeight, err := committer.commitBlockPair(ctx, bp)
		if err != nil {
			return 0, err
		}

		// if receiving service keep requesting the current height we are stuck
		if i == nextHeight {
			return 0, fmt.Errorf("failed to sync block at height %d", i)
		}
		i = nextHeight
	}

	return topBlockHeight, nil
}

func NewServiceBlockSync(ctx context.Context, logger log.BasicLogger, source blockSource, committer BlockPairCommitter) {
	ctx = trace.NewContext(ctx, committer.getServiceName())
	logger = logger.WithTags(trace.LogFieldFrom(ctx))
	supervised.GoForever(ctx, logger, func() {

		var height primitives.BlockHeight
		var err error
		for  err == nil {
			err = source.GetBlockTracker().WaitForBlock(ctx, height + 1)
			if err != nil {
				logger.Info("failed waiting for block", log.Error(err))
				return
			}
			height, err = syncOnce(ctx, source, committer, logger)
		}
	})
}
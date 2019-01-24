package testkit

import (
	"context"
	"errors"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter/memory"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type TamperingInMemoryBlockPersistence interface {
	adapter.BlockPersistence
	FailNextBlocks()
	WaitForTransaction(ctx context.Context, txHash primitives.Sha256) primitives.BlockHeight
}

func NewBlockPersistence(parent log.BasicLogger, preloadedBlocks []*protocol.BlockPairContainer, metricFactory metric.Factory) *tamperingBlockPersistence {
	logger := parent.WithTags(log.String("adapter", "block-storage"))
	return &tamperingBlockPersistence{
		InMemoryBlockPersistence: *memory.NewBlockPersistence(logger, metricFactory, preloadedBlocks...),
		txRegistry:               newTxWaiter(logger, preloadedBlocks),
	}
}

type tamperingBlockPersistence struct {
	memory.InMemoryBlockPersistence
	failNextBlocks bool

	txRegistry *txWaiter
}

func (bp *tamperingBlockPersistence) WaitForTransaction(ctx context.Context, txHash primitives.Sha256) primitives.BlockHeight {
	return bp.txRegistry.waitForTransaction(ctx, txHash)
}

func (bp *tamperingBlockPersistence) FailNextBlocks() {
	bp.failNextBlocks = true
}

func (bp *tamperingBlockPersistence) WriteNextBlock(blockPair *protocol.BlockPairContainer) error {
	if bp.failNextBlocks {
		return errors.New("could not write a block")
	}
	err := bp.InMemoryBlockPersistence.WriteNextBlock(blockPair)
	if err != nil {
		return err
	}
	bp.advertiseAllTransactions(blockPair.TransactionsBlock)
	return nil
}

func (bp *tamperingBlockPersistence) advertiseAllTransactions(block *protocol.TransactionsBlockContainer) {
	bp.txRegistry.advertiseTransactions(block.Header.BlockHeight(), block.SignedTransactions)
}

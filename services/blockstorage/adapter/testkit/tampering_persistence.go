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

type tamperingBlockPersistence struct {
	memory.InMemoryBlockPersistence
	failNextBlocks bool

	txTracker *txTracker
}

func NewBlockPersistence(parent log.BasicLogger, preloadedBlocks []*protocol.BlockPairContainer, metricFactory metric.Factory) *tamperingBlockPersistence {
	logger := parent.WithTags(log.String("adapter", "block-storage"))
	return &tamperingBlockPersistence{
		InMemoryBlockPersistence: *memory.NewBlockPersistence(logger, metricFactory, preloadedBlocks...),
		txTracker:                newTxTracker(logger, preloadedBlocks),
	}
}

func (bp *tamperingBlockPersistence) WaitForTransaction(ctx context.Context, txHash primitives.Sha256) primitives.BlockHeight {
	return bp.txTracker.waitForTransaction(ctx, txHash)
}

func (bp *tamperingBlockPersistence) FailNextBlocks() {
	bp.failNextBlocks = true
}

func (bp *tamperingBlockPersistence) WriteNextBlock(blockPair *protocol.BlockPairContainer) (bool, error) {
	if bp.failNextBlocks {
		return false, errors.New("could not write a block")
	}
	added, err := bp.InMemoryBlockPersistence.WriteNextBlock(blockPair)
	if err != nil {
		return added, err
	}
	if added {
		bp.advertiseAllTransactions(blockPair.TransactionsBlock)
	}
	return added, nil
}

func (bp *tamperingBlockPersistence) advertiseAllTransactions(block *protocol.TransactionsBlockContainer) {
	bp.txTracker.advertise(block.Header.BlockHeight(), block.SignedTransactions)
}

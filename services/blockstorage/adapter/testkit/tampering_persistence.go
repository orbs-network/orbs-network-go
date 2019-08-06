// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package testkit

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter/memory"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
)

type TamperingInMemoryBlockPersistence interface {
	adapter.BlockPersistence
	ResetTampering()
	TamperWithBlockWrites(maybeWriteFailedBlocks chan<- *protocol.BlockPairContainer)
	WaitForTransaction(ctx context.Context, txHash primitives.Sha256) primitives.BlockHeight
}

type tamperingBlockPersistence struct {
	memory.InMemoryBlockPersistence
	writeTamperingEnabled bool
	maybeTamperedBlocks   chan<- *protocol.BlockPairContainer

	txTracker *txTracker
}

func NewBlockPersistence(parent log.Logger, preloadedBlocks []*protocol.BlockPairContainer, metricFactory metric.Factory) *tamperingBlockPersistence {
	logger := parent.WithTags(log.String("adapter", "block-storage"))
	return &tamperingBlockPersistence{
		InMemoryBlockPersistence: *memory.NewBlockPersistence(logger, metricFactory, preloadedBlocks...),
		txTracker:                newTxTracker(logger, preloadedBlocks),
	}
}

func (bp *tamperingBlockPersistence) WaitForTransaction(ctx context.Context, txHash primitives.Sha256) primitives.BlockHeight {
	return bp.txTracker.waitForTransaction(ctx, txHash)
}

func (bp *tamperingBlockPersistence) ResetTampering() {
	bp.writeTamperingEnabled = false
	bp.maybeTamperedBlocks = nil
}

func (bp *tamperingBlockPersistence) TamperWithBlockWrites(maybeTamperedBlocks chan<- *protocol.BlockPairContainer) {
	bp.maybeTamperedBlocks = maybeTamperedBlocks
	bp.writeTamperingEnabled = true
}

func (bp *tamperingBlockPersistence) WriteNextBlock(blockPair *protocol.BlockPairContainer) (bool, primitives.BlockHeight, error) {
	if bp.writeTamperingEnabled {
		if mtb := bp.maybeTamperedBlocks; mtb != nil {
			mtb <- blockPair
		}
		return false, 0, errors.Errorf("intentionally failing (tampering with) WriteNextBlock() height %d", blockPair.ResultsBlock.Header.BlockHeight())
	}

	added, pHeight, err := bp.InMemoryBlockPersistence.WriteNextBlock(blockPair)
	if err != nil {
		return added, pHeight, err
	}

	if added {
		bp.advertiseAllTransactions(blockPair.TransactionsBlock)
	}
	return added, pHeight, nil
}

func (bp *tamperingBlockPersistence) advertiseAllTransactions(block *protocol.TransactionsBlockContainer) {
	bp.txTracker.advertise(block.Header.BlockHeight(), block.SignedTransactions)
}

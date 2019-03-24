// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package testkit

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"math"
	"sync"
)

type txTracker struct {
	sync.Mutex
	txToHeight   map[string]primitives.BlockHeight
	topHeight    primitives.BlockHeight
	blockTracker *synchronization.BlockTracker
	logger       log.BasicLogger
}

// this struct will leak memory in production. intended for use only in short lived tests
func newTxTracker(logger log.BasicLogger, preloadedBlocks []*protocol.BlockPairContainer) *txTracker {
	tracker := &txTracker{
		Mutex:        sync.Mutex{},
		txToHeight:   make(map[string]primitives.BlockHeight),
		blockTracker: synchronization.NewBlockTracker(logger, 0, math.MaxUint16),
		logger:       logger,
	}

	for _, bpc := range preloadedBlocks {
		tracker.advertise(bpc.TransactionsBlock.Header.BlockHeight(), bpc.TransactionsBlock.SignedTransactions)
	}

	return tracker
}

func (t *txTracker) getBlockHeight(txHash primitives.Sha256) (primitives.BlockHeight, primitives.BlockHeight) {
	t.Lock()
	defer t.Unlock()

	return t.txToHeight[txHash.KeyForMap()], t.topHeight
}

func (t *txTracker) advertise(height primitives.BlockHeight, transactions []*protocol.SignedTransaction) {
	if height == 0 {
		panic("illegal block height 0")
	}

	t.Lock()
	defer t.Unlock()

	if height <= t.topHeight { // block already advertised
		t.logger.Info("advertising block transactions aborted - already advertised", log.BlockHeight(height))
		return
	}

	for _, tx := range transactions {
		txHash := digest.CalcTxHash(tx.Transaction())

		prevHeight, existed := t.txToHeight[txHash.KeyForMap()]

		if existed {
			if prevHeight != height {
				t.logger.Error("FORK/DOUBLE-SPEND!! same transaction reported in different heights. may be committed twice", log.Transaction(txHash), log.BlockHeight(height), log.Uint64("previously-reported-height", uint64(prevHeight)))
				panic(fmt.Sprintf("FORK/DOUBLE-SPEND!! transaction %s previously advertised for height %d and now again for height %d. may be committed twice", txHash.String(), prevHeight, height))
			} else {
				t.logger.Error("BUG!! txTracker.txToHeight contains a block height ahead of topHeight", log.Transaction(txHash), log.BlockHeight(height), log.Uint64("tracker-top-height", uint64(t.topHeight)))
				panic(fmt.Sprintf("BUG!! txTracker.txToHeight contains a block height ahead of topHeight. tx %s found listed for height %d. but topHeight is %d", txHash.String(), height, t.topHeight))
			}
		}

		t.txToHeight[txHash.KeyForMap()] = height
	}
	t.logger.Info("advertising block transactions done", log.BlockHeight(height))

	t.blockTracker.IncrementTo(height)
	t.topHeight = height
}

func (t *txTracker) waitForTransaction(ctx context.Context, txHash primitives.Sha256) primitives.BlockHeight {
	logger := t.logger.WithTags(trace.LogFieldFrom(ctx))
	logger.Info("waiting for transaction", log.Transaction(txHash))
	for {
		txHeight, topHeight := t.getBlockHeight(txHash)

		if txHeight > 0 { // found requested height
			logger.Info("transaction found in block", log.Transaction(txHash), log.BlockHeight(txHeight))
			return txHeight
		}

		logger.Info("transaction not found in current block, will wait for next block to look for it again", log.Transaction(txHash), log.BlockHeight(topHeight))
		err := t.blockTracker.WaitForBlock(ctx, topHeight+1) // wait for next block
		if err != nil {
			instrumentation.DebugPrintGoroutineStacks(logger) // since test timed out, help find deadlocked goroutines
			panic(fmt.Sprintf("timed out waiting for transaction with hash %s", txHash))
		}
	}
}

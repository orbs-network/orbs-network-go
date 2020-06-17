// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package testkit

import (
	"context"
	"fmt"
	"github.com/orbs-network/crypto-lib-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/scribe/log"
	"math"
	"sync"
	"time"
)

const WaitForTransactionInterval = 5 * time.Millisecond

type txTracker struct {
	sync.Mutex
	txToHeight   map[string]primitives.BlockHeight
	topHeight    primitives.BlockHeight
	blockTracker *synchronization.BlockTracker
	logger       log.Logger
}

// this struct will leak memory in production. intended for use only in short lived tests
func newTxTracker(logger log.Logger, preloadedBlocks []*protocol.BlockPairContainer) *txTracker {
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

	for _, tx := range transactions {
		txHash := digest.CalcTxHash(tx.Transaction())

		_, existed := t.txToHeight[txHash.KeyForMap()]

		if !existed {
			t.txToHeight[txHash.KeyForMap()] = height
		}
	}
	t.logger.Info("advertising block transactions done", logfields.BlockHeight(height))

	t.topHeight = height
}

func (t *txTracker) waitForTransaction(ctx context.Context, txHash primitives.Sha256) primitives.BlockHeight {
	logger := t.logger.WithTags(trace.LogFieldFrom(ctx))
	logger.Info("waiting for transaction", logfields.Transaction(txHash))
	for {
		if ctx.Err() != nil {
			instrumentation.DebugPrintGoroutineStacks(logger) // since test timed out, help find deadlocked goroutines
			logger.Error(fmt.Sprintf("timed out waiting for transaction with hash %s", txHash))
			return 0
		}
		txHeight, topHeight := t.getBlockHeight(txHash)

		if txHeight > 0 { // found requested height
			logger.Info("transaction found in block", logfields.Transaction(txHash), logfields.BlockHeight(txHeight))
			return txHeight
		}

		logger.Info("transaction not found in current block, will wait for next block to look for it again", logfields.Transaction(txHash), logfields.BlockHeight(topHeight))
		time.Sleep(WaitForTransactionInterval)
	}
}

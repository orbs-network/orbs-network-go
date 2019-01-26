package testkit

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"math"
	"sync"
)

type txWaiter struct {
	sync.Mutex
	txToHeight     map[string]primitives.BlockHeight
	topKnownHeight primitives.BlockHeight
	idxTracker     *synchronization.BlockTracker
	parent         log.BasicLogger
}

func newTxWaiter(logger log.BasicLogger, preloadedBlocks []*protocol.BlockPairContainer) *txWaiter {
	registry := &txWaiter{
		Mutex:      sync.Mutex{},
		txToHeight: make(map[string]primitives.BlockHeight),
		idxTracker: synchronization.NewBlockTracker(logger, 0, math.MaxUint16),
		parent:     logger,
	}

	for _, bpc := range preloadedBlocks {
		registry.advertiseTransactions(bpc.TransactionsBlock.Header.BlockHeight(), bpc.TransactionsBlock.SignedTransactions)
	}

	return registry
}

func (txi *txWaiter) getBlockHeight(txHash primitives.Sha256) (primitives.BlockHeight, primitives.BlockHeight) {
	txi.Lock()
	defer txi.Unlock()

	return txi.txToHeight[txHash.KeyForMap()], txi.topKnownHeight
}

func (txi *txWaiter) advertiseTransactions(height primitives.BlockHeight, transactions []*protocol.SignedTransaction) {
	if height == 0 {
		panic("illegal block height 0")
	}

	txi.Lock()
	defer txi.Unlock()

	if height <= txi.topKnownHeight { // block already advertised
		txi.parent.Info("advertising block transactions aborted - already advertised", log.BlockHeight(height))
		return
	}

	for _, tx := range transactions {
		txHash := digest.CalcTxHash(tx.Transaction())

		prevHeight, existed := txi.txToHeight[txHash.KeyForMap()]

		if existed {
			assertSameHeight(prevHeight, height, txi.parent, txHash)
			panic(fmt.Sprintf("BUG!! txWaiter.txToHeight contains a block height ahead of topKnownHeight. tx %s found listed for height %d. but topKnownHeight is %d", txHash.String(), height, txi.topKnownHeight))
		}

		txi.txToHeight[txHash.KeyForMap()] = height
	}
	txi.parent.Info("advertising block transactions done", log.BlockHeight(height))

	txi.idxTracker.IncrementTo(height)
	txi.topKnownHeight = height
}

func (txi *txWaiter) waitForTransaction(ctx context.Context, txHash primitives.Sha256) primitives.BlockHeight {
	logger := txi.parent.WithTags(trace.LogFieldFrom(ctx))
	logger.Info("waiting for transaction", log.Transaction(txHash))
	for {
		txHeight, topHeight := txi.getBlockHeight(txHash)

		if txHeight > 0 { // found requested height
			logger.Info("transaction found in block", log.Transaction(txHash), log.BlockHeight(txHeight))
			return txHeight
		}

		logger.Info("transaction not found as of block", log.Transaction(txHash), log.BlockHeight(topHeight))
		err := txi.idxTracker.WaitForBlock(ctx, topHeight+1) // wait for next block
		if err != nil {
			test.DebugPrintGoroutineStacks() // since test timed out, help find deadlocked goroutines
			panic(fmt.Sprintf("timed out waiting for transaction with hash %s", txHash))
		}
	}
}

func assertSameHeight(prevHeight primitives.BlockHeight, newHeight primitives.BlockHeight, logger log.BasicLogger, txHash primitives.Sha256) {
	if prevHeight != newHeight {
		logger.Error("FORK/DOUBLE-SPEND!!!! same transaction reported in different heights. may be committed twice", log.Transaction(txHash), log.BlockHeight(newHeight), log.Uint64("previously-reported-height", uint64(prevHeight)))
		panic(fmt.Sprintf("FORK/DOUBLE-SPEND!!!! transaction %s previously advertised for height %d and now again for height %d. may be committed twice", txHash.String(), prevHeight, newHeight))
	}
}

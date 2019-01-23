package testkit

import (
	"context"
	"errors"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter/memory"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"sync"
)

type blockHeightChan struct {
	c chan struct{}
	h primitives.BlockHeight
}

type TamperingInMemoryBlockPersistence interface {
	adapter.BlockPersistence
	FailNextBlocks()
	WaitForTransaction(ctx context.Context, txHash primitives.Sha256) primitives.BlockHeight
}

func NewBlockPersistence(parent log.BasicLogger, preloadedBlocks []*protocol.BlockPairContainer, metricFactory metric.Factory) *tamperingBlockPersistence {
	logger := parent.WithTags(log.String("adapter", "block-storage"))
	p := &tamperingBlockPersistence{
		InMemoryBlockPersistence: *memory.NewBlockPersistence(logger, metricFactory, preloadedBlocks...),
	}

	p.txToBlockHeightChan.channels = make(map[string]*blockHeightChan)
	for _, bpc := range preloadedBlocks {
		p.advertiseAllTransactions(bpc.TransactionsBlock)
	}

	return p
}

type tamperingBlockPersistence struct {
	memory.InMemoryBlockPersistence
	failNextBlocks bool

	txToBlockHeightChan struct {
		sync.Mutex
		channels map[string]*blockHeightChan
	}
}

func (bp *tamperingBlockPersistence) WaitForTransaction(ctx context.Context, txHash primitives.Sha256) primitives.BlockHeight {
	bhc := bp.getChanForTxHash(txHash)

	select {
	case <-bhc.c: // when c closes, h will already be written
		return bhc.h
	case <-ctx.Done():
		test.DebugPrintGoroutineStacks() // since test timed out, help find deadlocked goroutines
		panic(fmt.Sprintf("timed out waiting for transaction with hash %s", txHash))
	}
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
	bp.txToBlockHeightChan.Lock()
	defer bp.txToBlockHeightChan.Unlock()

	height := block.Header.BlockHeight()
	if height == 0 {
		panic("illegal block h 0")
	}

	for _, tx := range block.SignedTransactions {
		txHash := digest.CalcTxHash(tx.Transaction())

		bhc := bp.getChanForTxHashUnlocked(txHash)

		if bhc.h != 0 { // previously advertised
			checkForConflicts(bhc.h, height, bp.Logger, txHash)
			continue
		}

		bhc.h = height // first
		close(bhc.c)   // second
		bp.Logger.Info("advertising transaction completion", log.Transaction(txHash), log.BlockHeight(height))
	}
}

func checkForConflicts(prevHeight primitives.BlockHeight, newHeight primitives.BlockHeight, logger log.BasicLogger, txHash primitives.Sha256) {
	if prevHeight != newHeight {
		logger.Error("FORK/DOUBLE-SPEND!!!! same transaction reported in different heights. may be committed twice", log.Transaction(txHash), log.BlockHeight(newHeight), log.Uint64("previously-reported-height", uint64(prevHeight)))
		panic(fmt.Sprintf("FORK/DOUBLE-SPEND!!!! transaction %s previously advertised for height %d and now again for height %d. may be committed twice", txHash.String(), prevHeight, newHeight))
	}
	logger.Info("advertising transaction completion aborted - already advertised", log.Transaction(txHash), log.BlockHeight(newHeight))
}

func (bp *tamperingBlockPersistence) getChanForTxHash(txHash primitives.Sha256) *blockHeightChan {
	bp.txToBlockHeightChan.Lock()
	defer bp.txToBlockHeightChan.Unlock()

	bhc := bp.getChanForTxHashUnlocked(txHash)

	return bhc
}

func (bp *tamperingBlockPersistence) getChanForTxHashUnlocked(txHash primitives.Sha256) *blockHeightChan {
	bhc := bp.txToBlockHeightChan.channels[txHash.KeyForMap()]
	if bhc == nil {
		bhc = &blockHeightChan{
			c: make(chan struct{}),
		}
		bp.txToBlockHeightChan.channels[txHash.KeyForMap()] = bhc
	}
	return bhc
}

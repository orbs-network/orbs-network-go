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

type blockHeightChan chan primitives.BlockHeight

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

	p.blockHeightsPerTxHash.channels = make(map[string]blockHeightChan)
	for _, bpc := range preloadedBlocks {
		p.advertiseAllTransactions(bpc.TransactionsBlock)
	}

	return p
}

type tamperingBlockPersistence struct {
	memory.InMemoryBlockPersistence
	failNextBlocks bool

	blockHeightsPerTxHash struct {
		sync.Mutex
		channels map[string]blockHeightChan
	}
}

func (bp *tamperingBlockPersistence) WaitForTransaction(ctx context.Context, txHash primitives.Sha256) primitives.BlockHeight {
	ch := bp.getChanFor(txHash)

	select {
	case h := <-ch:
		return h
	case <-ctx.Done():
		bp.Logger.Info("tamperingBlockPersistence context terminated panic", log.Transaction(txHash))
		test.DebugPrintGoroutineStacks() // since test timed out, help find deadlocked goroutines
		// TODO https://github.com/orbs-network/orbs-network-go/issues/785 remove this panic()! logger prints with delay, but panic() prints immediately so it's out of context when reading logs (as if it's printed from a future test which the log hasn't printed yet)
		panic(fmt.Sprintf("context terminated while waiting for transaction with hash %s", txHash))
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
	for _, tx := range block.SignedTransactions {
		notified := bp.notifyChanFor(digest.CalcTxHash(tx.Transaction()), block.Header.BlockHeight())
		if notified == false {
			bp.Logger.Info("advertising transaction completion already occurred", log.Transaction(digest.CalcTxHash(tx.Transaction())), log.BlockHeight(block.Header.BlockHeight()))
			continue
		}
		bp.Logger.Info("advertising transaction completion", log.Transaction(digest.CalcTxHash(tx.Transaction())), log.BlockHeight(block.Header.BlockHeight()))
	}
}

func (bp *tamperingBlockPersistence) notifyChanFor(txHash primitives.Sha256, height primitives.BlockHeight) (notified bool) {
	defer func() {
		recover() //BlockPersistence.WriteNextBlock() does not return "added" so it's possible to notifyChanFor() twice
	}()
	ch := bp.getChanFor(txHash)
	ch <- height
	close(ch)
	notified = true // reach here only if ch was open - first invocation for txHash
	return
}

func (bp *tamperingBlockPersistence) getChanFor(txHash primitives.Sha256) blockHeightChan {
	bp.blockHeightsPerTxHash.Lock()
	defer bp.blockHeightsPerTxHash.Unlock()

	ch, ok := bp.blockHeightsPerTxHash.channels[txHash.KeyForMap()]
	if !ok {
		ch = make(blockHeightChan, 1)
		bp.blockHeightsPerTxHash.channels[txHash.KeyForMap()] = ch
	}

	return ch
}

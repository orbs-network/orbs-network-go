package adapter

import (
	"context"
	"errors"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"sync"
)

type TamperingInMemoryBlockPersistence interface {
	BlockPersistence
	FailNextBlocks()
	WaitForTransaction(ctx context.Context, txHash primitives.Sha256) primitives.BlockHeight
}

func NewTamperingInMemoryBlockPersistence(parent log.BasicLogger, preloadedBlocks []*protocol.BlockPairContainer, metricFactory metric.Factory) *tamperingBlockPersistence {
	logger := parent.WithTags(log.String("adapter", "block-storage"))
	p := &tamperingBlockPersistence{
		InMemoryBlockPersistence: InMemoryBlockPersistence{
			logger:     logger,
			metrics:    &memMetrics{size: metricFactory.NewGauge("BlockStorage.InMemoryBlockPersistence.SizeInBytes")},
			tracker:    synchronization.NewBlockTracker(logger, uint64(len(preloadedBlocks)), 5),
			blockChain: aChainOfBlocks{blocks: preloadedBlocks},
		},
	}

	p.blockHeightsPerTxHash.channels = make(map[string]blockHeightChan)
	for _, bpc := range preloadedBlocks {
		p.advertiseAllTransactions(bpc.TransactionsBlock)
	}

	return p
}

type tamperingBlockPersistence struct {
	InMemoryBlockPersistence
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
	for _, tx := range block.SignedTransactions {
		txHash := digest.CalcTxHash(tx.Transaction())
		bp.logger.Info("advertising transaction completion", log.Transaction(txHash), log.BlockHeight(block.Header.BlockHeight()))
		ch := bp.getChanFor(txHash)
		ch <- block.Header.BlockHeight() // this will panic with "send on closed channel" if the same tx is added twice to blocks (duplicate tx hash!!)
		close(ch)
	}
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

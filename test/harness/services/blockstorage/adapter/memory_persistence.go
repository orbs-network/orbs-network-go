package adapter

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"sync"
	"time"
	"unsafe"
)

type InMemoryBlockPersistence interface {
	adapter.BlockPersistence
	FailNextBlocks()
	WaitForTransaction(ctx context.Context, txhash primitives.Sha256) primitives.BlockHeight
}

type blockHeightChan chan primitives.BlockHeight

type metrics struct {
	size *metric.Gauge
}

type inMemoryBlockPersistence struct {
	blockChain struct {
		sync.RWMutex
		blocks []*protocol.BlockPairContainer
	}

	failNextBlocks bool
	tracker        *synchronization.BlockTracker
	logger         log.BasicLogger

	blockHeightsPerTxHash struct {
		sync.Mutex
		channels map[string]blockHeightChan
	}

	metrics *metrics
}

func newMetrics(m metric.Factory) *metrics {
	return &metrics{
		size: m.NewGauge("BlockStorage.InMemoryBlockPersistence.SizeInMB"),
	}
}

func NewInMemoryBlockPersistence(parent log.BasicLogger, metricFactory metric.Factory) InMemoryBlockPersistence {
	logger := parent.WithTags(log.String("adapter", "block-storage"))
	p := &inMemoryBlockPersistence{
		failNextBlocks: false,
		logger:         logger,
		tracker:        synchronization.NewBlockTracker(0, 5),
		metrics:        newMetrics(metricFactory),
	}

	p.blockHeightsPerTxHash.channels = make(map[string]blockHeightChan)

	return p
}

func (bp *inMemoryBlockPersistence) GetBlockTracker() *synchronization.BlockTracker {
	return bp.tracker
}

func (bp *inMemoryBlockPersistence) WaitForTransaction(ctx context.Context, txHash primitives.Sha256) primitives.BlockHeight {
	ch := bp.getChanFor(txHash)

	select {
	case h := <-ch:
		return h
	case <-ctx.Done():
		test.DebugPrintGoroutineStacks() // since test timed out, help find deadlocked goroutines
		panic(fmt.Sprintf("timed out waiting for transaction with hash %s", txHash))
	}
}

func (bp *inMemoryBlockPersistence) GetLastBlock() (*protocol.BlockPairContainer, error) {
	bp.blockChain.RLock()
	defer bp.blockChain.RUnlock()

	count := len(bp.blockChain.blocks)
	if count == 0 {
		return nil, nil
	}

	return bp.blockChain.blocks[count-1], nil
}

func (bp *inMemoryBlockPersistence) GetNumBlocks() (primitives.BlockHeight, error) {
	bp.blockChain.RLock()
	defer bp.blockChain.RUnlock()

	return primitives.BlockHeight(len(bp.blockChain.blocks)), nil
}

func (bp *inMemoryBlockPersistence) WriteNextBlock(blockPair *protocol.BlockPairContainer) (bool, error) {
	if bp.failNextBlocks {
		return false, errors.New("could not write a block")
	}

	added, err := bp.validateAndAddNextBlock(blockPair)
	if err != nil || !added {
		return false, err
	}

	bp.tracker.IncrementHeight()
	bp.metrics.size.Add(sizeOfBlock(blockPair))

	bp.advertiseAllTransactions(blockPair.TransactionsBlock)

	return true, nil
}

func (bp *inMemoryBlockPersistence) validateAndAddNextBlock(blockPair *protocol.BlockPairContainer) (bool, error) {
	bp.blockChain.Lock()
	defer bp.blockChain.Unlock()

	if primitives.BlockHeight(len(bp.blockChain.blocks))+1 < blockPair.TransactionsBlock.Header.BlockHeight() {
		return false, errors.Errorf("block persistence tried to write next block with height %d when %d exist", blockPair.TransactionsBlock.Header.BlockHeight(), len(bp.blockChain.blocks))
	}

	if primitives.BlockHeight(len(bp.blockChain.blocks))+1 > blockPair.TransactionsBlock.Header.BlockHeight() {
		bp.logger.Info("block persistence ignoring write next block with height %d when %d exist", log.Uint64("incoming-block-height", uint64(blockPair.TransactionsBlock.Header.BlockHeight())), log.BlockHeight(primitives.BlockHeight(len(bp.blockChain.blocks))))
		return false, nil
	}
	bp.blockChain.blocks = append(bp.blockChain.blocks, blockPair)
	return true, nil
}

func (bp *inMemoryBlockPersistence) GetBlocksRelevantToTxTimestamp(txTimeStamp primitives.TimestampNano, rules adapter.BlockSearchRules) []*protocol.BlockPairContainer {
	start := txTimeStamp - primitives.TimestampNano(rules.StartGraceNano)
	end := txTimeStamp + primitives.TimestampNano(rules.EndGraceNano+rules.TransactionExpireNano)

	if end < start {
		return nil
	}
	var relevantBlocks []*protocol.BlockPairContainer
	interval := end - start
	// TODO: FIXME: sanity check, this is really useless here right now, but we are going to refactor this in about two-three weeks, and when we do, this is here to remind us to have a sanity check on this query
	if interval > primitives.TimestampNano(time.Hour.Nanoseconds()) {
		return nil
	}

	bp.blockChain.RLock()
	defer bp.blockChain.RUnlock()

	blockPairs := bp.blockChain.blocks

	for _, blockPair := range blockPairs {
		delta := end - blockPair.TransactionsBlock.Header.Timestamp()
		if delta > 0 && interval > delta {
			relevantBlocks = append(relevantBlocks, blockPair)
		}
	}
	return relevantBlocks
}

func (bp *inMemoryBlockPersistence) getBlockPairAtHeight(height primitives.BlockHeight) (*protocol.BlockPairContainer, error) {
	bp.blockChain.RLock()
	defer bp.blockChain.RUnlock()

	if height > primitives.BlockHeight(len(bp.blockChain.blocks)) {
		return nil, errors.Errorf("block with height %d not found in block persistence", height)
	}

	return bp.blockChain.blocks[height-1], nil
}

func (bp *inMemoryBlockPersistence) GetTransactionsBlock(height primitives.BlockHeight) (*protocol.TransactionsBlockContainer, error) {
	blockPair, err := bp.getBlockPairAtHeight(height)
	if err != nil {
		return nil, err
	}
	return blockPair.TransactionsBlock, nil
}

func (bp *inMemoryBlockPersistence) GetResultsBlock(height primitives.BlockHeight) (*protocol.ResultsBlockContainer, error) {
	blockPair, err := bp.getBlockPairAtHeight(height)
	if err != nil {
		return nil, err
	}
	return blockPair.ResultsBlock, nil
}

func (bp *inMemoryBlockPersistence) FailNextBlocks() {
	bp.failNextBlocks = true
}

// Is covered by the mutex in WriteNextBlock
func (bp *inMemoryBlockPersistence) getChanFor(txHash primitives.Sha256) blockHeightChan {
	bp.blockHeightsPerTxHash.Lock()
	defer bp.blockHeightsPerTxHash.Unlock()

	ch, ok := bp.blockHeightsPerTxHash.channels[txHash.KeyForMap()]
	if !ok {
		ch = make(blockHeightChan, 1)
		bp.blockHeightsPerTxHash.channels[txHash.KeyForMap()] = ch
	}

	return ch
}

func (bp *inMemoryBlockPersistence) advertiseAllTransactions(block *protocol.TransactionsBlockContainer) {
	for _, tx := range block.SignedTransactions {
		txHash := digest.CalcTxHash(tx.Transaction())
		bp.logger.Info("advertising transaction completion", log.Transaction(txHash), log.BlockHeight(block.Header.BlockHeight()))
		ch := bp.getChanFor(txHash)
		ch <- block.Header.BlockHeight() // this will panic with "send on closed channel" if the same tx is added twice to blocks (duplicate tx hash!!)
		close(ch)
	}
}

// TODO: better support for paging
func (bp *inMemoryBlockPersistence) GetBlocks(first primitives.BlockHeight, last primitives.BlockHeight) (blocks []*protocol.BlockPairContainer, firstReturnedBlockHeight primitives.BlockHeight, lastReturnedBlockHeight primitives.BlockHeight, err error) {
	// FIXME use more efficient way to slice blocks

	bp.blockChain.RLock()
	defer bp.blockChain.RUnlock()

	allBlocks := bp.blockChain.blocks
	allBlocksLength := primitives.BlockHeight(len(allBlocks))

	if first > allBlocksLength {
		return nil, 0, 0, nil
	}
	firstReturnedBlockHeight = first

	lastReturnedBlockHeight = last
	if last > allBlocksLength {
		lastReturnedBlockHeight = allBlocksLength
	}

	for i := first - 1; i < lastReturnedBlockHeight; i++ {
		blocks = append(blocks, allBlocks[i])
	}

	return blocks, firstReturnedBlockHeight, lastReturnedBlockHeight, nil
}

func sizeOfBlock(block *protocol.BlockPairContainer) int64 {
	txBlock := block.TransactionsBlock
	txBlockSize := len(txBlock.Header.Raw()) + len(txBlock.BlockProof.Raw()) + len(txBlock.Metadata.Raw())

	rsBlock := block.ResultsBlock
	rsBlockSize := len(rsBlock.Header.Raw()) + len(rsBlock.BlockProof.Raw())

	txBlockPointers := unsafe.Sizeof(txBlock) + unsafe.Sizeof(txBlock.Header) + unsafe.Sizeof(txBlock.Metadata) + unsafe.Sizeof(txBlock.BlockProof) + unsafe.Sizeof(txBlock.SignedTransactions)
	rsBlockPointers := unsafe.Sizeof(rsBlock) + unsafe.Sizeof(rsBlock.Header) + unsafe.Sizeof(rsBlock.BlockProof) + unsafe.Sizeof(rsBlock.TransactionReceipts) + unsafe.Sizeof(rsBlock.ContractStateDiffs)

	for _, tx := range txBlock.SignedTransactions {
		txBlockSize += len(tx.Raw())
		txBlockPointers += unsafe.Sizeof(tx)
	}

	for _, diff := range rsBlock.ContractStateDiffs {
		rsBlockSize += len(diff.Raw())
		rsBlockPointers += unsafe.Sizeof(diff)
	}

	for _, receipt := range rsBlock.TransactionReceipts {
		rsBlockSize += len(receipt.Raw())
		rsBlockPointers += unsafe.Sizeof(receipt)
	}

	pointers := unsafe.Sizeof(block) + txBlockPointers + rsBlockPointers

	return int64(txBlockSize) + int64(rsBlockSize) + int64(pointers)
}

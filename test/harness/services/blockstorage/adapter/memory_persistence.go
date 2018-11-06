package adapter

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"sync"
	"time"
)

type InMemoryBlockPersistence interface {
	adapter.BlockPersistence
	FailNextBlocks()
	WaitForTransaction(ctx context.Context, txhash primitives.Sha256) primitives.BlockHeight
}

type blockHeightChan chan primitives.BlockHeight

type inMemoryBlockPersistence struct {
	blockChain struct {
		sync.RWMutex
		blocks []*protocol.BlockPairContainer
	}

	failNextBlocks bool
	tracker        *synchronization.BlockTracker

	blockHeightsPerTxHash struct {
		sync.Mutex
		channels map[string]blockHeightChan
	}
}

func NewInMemoryBlockPersistence() InMemoryBlockPersistence {
	p := &inMemoryBlockPersistence{
		failNextBlocks: false,
		tracker:        synchronization.NewBlockTracker(0, 5),
	}

	p.blockHeightsPerTxHash.channels = make(map[string]blockHeightChan)

	return p
}

func (bp *inMemoryBlockPersistence) GetBlockTracker() *synchronization.BlockTracker {
	return bp.tracker
}

func (bp *inMemoryBlockPersistence) WaitForTransaction(ctx context.Context, txhash primitives.Sha256) primitives.BlockHeight {
	ch := bp.getChanFor(txhash)

	select {
	case h := <-ch:
		return h
	case <-ctx.Done():
		test.DebugPrintGoroutineStacks() // since test timed out, help find deadlocked goroutines
		panic(fmt.Sprintf("timed out waiting for transaction with hash %s", txhash))
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

func (bp *inMemoryBlockPersistence) WriteNextBlock(blockPair *protocol.BlockPairContainer) error {
	if bp.failNextBlocks {
		return errors.New("could not write a block")
	}

	err := bp.validateAndAddNextBlock(blockPair)
	if err != nil {
		return err
	}

	bp.tracker.IncrementHeight()

	bp.advertiseAllTransactions(blockPair.TransactionsBlock)

	return nil
}

func (bp *inMemoryBlockPersistence) validateAndAddNextBlock(blockPair *protocol.BlockPairContainer) error {
	bp.blockChain.Lock()
	defer bp.blockChain.Unlock()

	if primitives.BlockHeight(len(bp.blockChain.blocks))+1 != blockPair.TransactionsBlock.Header.BlockHeight() {
		return errors.Errorf("block persistence tried to write next block with height %d when %d exist", blockPair.TransactionsBlock.Header.BlockHeight(), len(bp.blockChain.blocks))
	}

	bp.blockChain.blocks = append(bp.blockChain.blocks, blockPair)
	return nil
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
func (bp *inMemoryBlockPersistence) getChanFor(txhash primitives.Sha256) blockHeightChan {
	bp.blockHeightsPerTxHash.Lock()
	defer bp.blockHeightsPerTxHash.Unlock()

	ch, ok := bp.blockHeightsPerTxHash.channels[txhash.KeyForMap()]
	if !ok {
		ch = make(blockHeightChan, 1)
		bp.blockHeightsPerTxHash.channels[txhash.KeyForMap()] = ch
	}

	return ch
}

func (bp *inMemoryBlockPersistence) advertiseAllTransactions(block *protocol.TransactionsBlockContainer) {
	for _, tx := range block.SignedTransactions {
		ch := bp.getChanFor(digest.CalcTxHash(tx.Transaction()))
		select {
		case ch <- block.Header.BlockHeight():
		default:
			// FIXME: this happens when two txid are in different blocks (or same block), this should never happen and we do not log it here (too low) and we also do not want to stop the loop (break/return error)
			continue
		}
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

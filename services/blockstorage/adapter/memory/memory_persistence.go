// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package memory

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"sync"
	"unsafe"
)

type memMetrics struct {
	size *metric.Gauge
}

type aChainOfBlocks struct {
	sync.RWMutex
	blocks []*protocol.BlockPairContainer
}

type InMemoryBlockPersistence struct {
	blockChain aChainOfBlocks

	tracker *synchronization.BlockTracker
	Logger  log.Logger

	metrics *memMetrics
}

func NewBlockPersistence(parent log.Logger, metricFactory metric.Factory, preloadedBlocks ...*protocol.BlockPairContainer) *InMemoryBlockPersistence {
	logger := parent.WithTags(log.String("adapter", "block-storage"))
	p := &InMemoryBlockPersistence{
		Logger:     logger,
		metrics:    &memMetrics{size: metricFactory.NewGauge("BlockStorage.InMemoryBlockPersistenceSize.Bytes")},
		tracker:    synchronization.NewBlockTracker(logger, uint64(len(preloadedBlocks)), 5),
		blockChain: aChainOfBlocks{blocks: clone(preloadedBlocks)},
	}

	return p
}

// this is needed so that each instance of BlockPersistence has its own copy of the block chain
func clone(blocks []*protocol.BlockPairContainer) (cloned []*protocol.BlockPairContainer) {
	cloned = make([]*protocol.BlockPairContainer, len(blocks))
	copy(cloned, blocks)
	return
}

func (bp *InMemoryBlockPersistence) GetBlockTracker() *synchronization.BlockTracker {
	return bp.tracker
}

func (bp *InMemoryBlockPersistence) GetLastBlock() (*protocol.BlockPairContainer, error) {
	bp.blockChain.RLock()
	defer bp.blockChain.RUnlock()

	count := len(bp.blockChain.blocks)
	if count == 0 {
		return nil, nil
	}

	return bp.blockChain.blocks[count-1], nil
}

func (bp *InMemoryBlockPersistence) GetLastBlockHeight() (primitives.BlockHeight, error) {
	bp.blockChain.RLock()
	defer bp.blockChain.RUnlock()

	return primitives.BlockHeight(len(bp.blockChain.blocks)), nil
}

func (bp *InMemoryBlockPersistence) WriteNextBlock(blockPair *protocol.BlockPairContainer) (bool, primitives.BlockHeight, error) {

	added, pHeight := bp.validateAndAddNextBlock(blockPair)

	if added {
		bp.metrics.size.Add(sizeOfBlock(blockPair))
	}

	return added, pHeight, nil
}

func (bp *InMemoryBlockPersistence) validateAndAddNextBlock(blockPair *protocol.BlockPairContainer) (bool, primitives.BlockHeight) {
	bp.blockChain.Lock()
	defer bp.blockChain.Unlock()

	currentTop := primitives.BlockHeight(len(bp.blockChain.blocks))

	if primitives.BlockHeight(len(bp.blockChain.blocks))+1 != blockPair.TransactionsBlock.Header.BlockHeight() {
		return false, currentTop
	}

	// XXX TODO - all instances of bp.blockChain share the same instance of bp.blockChain.blocks. they are initialized from "preloadedBlocks"
	bp.blockChain.blocks = append(bp.blockChain.blocks, blockPair)
	bp.tracker.IncrementTo(blockPair.ResultsBlock.Header.BlockHeight())
	return true, blockPair.ResultsBlock.Header.BlockHeight()
}

func (bp *InMemoryBlockPersistence) GetBlockByTx(txHash primitives.Sha256, minBlockTs primitives.TimestampNano, maxBlockTs primitives.TimestampNano) (*protocol.BlockPairContainer, int, error) {

	bp.blockChain.RLock()
	defer bp.blockChain.RUnlock()

	allBlocks := bp.blockChain.blocks
	var candidateBlocks []*protocol.BlockPairContainer
	for _, blockPair := range allBlocks {
		bts := blockPair.TransactionsBlock.Header.Timestamp()
		if maxBlockTs >= bts && minBlockTs <= bts {
			candidateBlocks = append(candidateBlocks, blockPair)
		}
	}

	if len(candidateBlocks) == 0 {
		return nil, 0, nil
	}

	for _, b := range candidateBlocks {
		for txi, txr := range b.ResultsBlock.TransactionReceipts {
			if txr.Txhash().Equal(txHash) {
				return b, txi, nil
			}
		}
	}
	return nil, 0, nil
}

func (bp *InMemoryBlockPersistence) getBlockPairAtHeight(height primitives.BlockHeight) (*protocol.BlockPairContainer, error) {
	bp.blockChain.RLock()
	defer bp.blockChain.RUnlock()

	if height > primitives.BlockHeight(len(bp.blockChain.blocks)) {
		return nil, errors.Errorf("block with height %d not found in block persistence", height)
	}

	return bp.blockChain.blocks[height-1], nil
}

func (bp *InMemoryBlockPersistence) GetTransactionsBlock(height primitives.BlockHeight) (*protocol.TransactionsBlockContainer, error) {
	blockPair, err := bp.getBlockPairAtHeight(height)
	if err != nil {
		return nil, err
	}
	return blockPair.TransactionsBlock, nil
}

func (bp *InMemoryBlockPersistence) GetResultsBlock(height primitives.BlockHeight) (*protocol.ResultsBlockContainer, error) {
	blockPair, err := bp.getBlockPairAtHeight(height)
	if err != nil {
		return nil, err
	}
	return blockPair.ResultsBlock, nil
}

func (bp *InMemoryBlockPersistence) ScanBlocks(from primitives.BlockHeight, pageSize uint8, f adapter.CursorFunc) error {
	bp.blockChain.RLock()
	defer bp.blockChain.RUnlock()

	allBlocks := bp.blockChain.blocks
	allBlocksLength := primitives.BlockHeight(len(allBlocks))

	if allBlocksLength < from || from == 0 {
		return fmt.Errorf("requested unknown block height %d. current height is %d", from, allBlocksLength)
	}

	wantsMore := true
	for from <= allBlocksLength && wantsMore {
		fromIndex := from - 1
		toIndex := fromIndex + primitives.BlockHeight(pageSize)
		if toIndex > allBlocksLength {
			toIndex = allBlocksLength
		}
		wantsMore = f(from, allBlocks[fromIndex:toIndex])
		from = toIndex + 1
	}
	return nil
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

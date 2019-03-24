// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package memory

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
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
	blockChain struct {
		sync.RWMutex
		blocks []*protocol.BlockPairContainer
	}

	tracker *synchronization.BlockTracker
	Logger  log.BasicLogger

	metrics *memMetrics
}

func NewBlockPersistence(parent log.BasicLogger, metricFactory metric.Factory, preloadedBlocks ...*protocol.BlockPairContainer) *InMemoryBlockPersistence {
	logger := parent.WithTags(log.String("adapter", "block-storage"))
	p := &InMemoryBlockPersistence{
		Logger:     logger,
		metrics:    &memMetrics{size: metricFactory.NewGauge("BlockStorage.InMemoryBlockPersistenceSize.Bytes")},
		tracker:    synchronization.NewBlockTracker(logger, uint64(len(preloadedBlocks)), 5),
		blockChain: aChainOfBlocks{blocks: preloadedBlocks},
	}

	return p
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

func (bp *InMemoryBlockPersistence) WriteNextBlock(blockPair *protocol.BlockPairContainer) (bool, error) {

	added, err := bp.validateAndAddNextBlock(blockPair)
	if err != nil || !added {
		return added, err
	}

	bp.metrics.size.Add(sizeOfBlock(blockPair))

	return added, nil
}

func (bp *InMemoryBlockPersistence) validateAndAddNextBlock(blockPair *protocol.BlockPairContainer) (bool, error) {
	bp.blockChain.Lock()
	defer bp.blockChain.Unlock()

	if !(blockPair.ResultsBlock.BlockProof.IsTypeBenchmarkConsensus() || blockPair.ResultsBlock.BlockProof.IsTypeLeanHelix()) {
		return false, errors.Errorf("block persistence tried to write block with invalid proof type: %s", blockPair.ResultsBlock.BlockProof)
	}

	if primitives.BlockHeight(len(bp.blockChain.blocks))+1 < blockPair.TransactionsBlock.Header.BlockHeight() {
		return false, errors.Errorf("block persistence tried to write next block with height %d when %d exist", blockPair.TransactionsBlock.Header.BlockHeight(), len(bp.blockChain.blocks))
	}

	if primitives.BlockHeight(len(bp.blockChain.blocks))+1 > blockPair.TransactionsBlock.Header.BlockHeight() {
		bp.Logger.Info("block persistence ignoring write next block. incorrect height", log.Uint64("incoming-block-height", uint64(blockPair.TransactionsBlock.Header.BlockHeight())), log.BlockHeight(primitives.BlockHeight(len(bp.blockChain.blocks))))
		// TODO(v1): since we're skipping the add, byte-compare the block header to make sure we don't have a fork
		return false, nil
	}
	bp.blockChain.blocks = append(bp.blockChain.blocks, blockPair)
	bp.tracker.IncrementTo(blockPair.ResultsBlock.Header.BlockHeight())
	return true, nil
}

func (bp *InMemoryBlockPersistence) GetBlockByTx(txHash primitives.Sha256, minBlockTs primitives.TimestampNano, maxBlockTs primitives.TimestampNano) (*protocol.BlockPairContainer, int, error) {

	bp.blockChain.RLock()
	defer bp.blockChain.RUnlock()

	allBlocks := bp.blockChain.blocks
	var candidateBlocks []*protocol.BlockPairContainer
	for _, blockPair := range allBlocks {
		bts := blockPair.TransactionsBlock.Header.Timestamp()
		if maxBlockTs > bts && minBlockTs < bts {
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

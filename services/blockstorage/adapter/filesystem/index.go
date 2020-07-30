// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package filesystem

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/internodesync"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/scribe/log"
	"sync"
)

type blockHeightIndex struct {
	sync.RWMutex
	heightOffset         map[primitives.BlockHeight]int64
	firstBlockInTsBucket map[uint32]primitives.BlockHeight
	nextOffset           int64
	sequentialTopBlock   *protocol.BlockPairContainer
	topBlock             *protocol.BlockPairContainer
	lastWrittenBlock     *protocol.BlockPairContainer
	logger               log.Logger
}

func newBlockHeightIndex(logger log.Logger, firstBlockOffset int64) *blockHeightIndex {
	return &blockHeightIndex{
		logger:               logger,
		heightOffset:         map[primitives.BlockHeight]int64{},
		firstBlockInTsBucket: map[uint32]primitives.BlockHeight{},
		nextOffset:           firstBlockOffset,
		sequentialTopBlock:   nil,
		topBlock:             nil,
		lastWrittenBlock:     nil,
	}
}

func (i *blockHeightIndex) getSyncState() internodesync.SyncState {
	i.RLock()
	defer i.RUnlock()
	return internodesync.SyncState{
		TopBlock:        i.topBlock,
		InOrderBlock:    i.sequentialTopBlock,
		LastSyncedBlock: i.lastWrittenBlock,
	}
}

func (i *blockHeightIndex) fetchNextOffset() int64 {
	i.RLock()
	defer i.RUnlock()

	return i.nextOffset
}

func (i *blockHeightIndex) fetchBlockOffset(height primitives.BlockHeight) (offset int64, ok bool) {
	i.RLock()
	defer i.RUnlock()

	offset, ok = i.heightOffset[height]
	return
}

// ignores blocks which are not fully synced (storage is missing blocks with lower height)
func (i *blockHeightIndex) getEarliestTxBlockInBucketForTsRange(rangeStart primitives.TimestampNano, rangeEnd primitives.TimestampNano) (primitives.BlockHeight, bool) {
	i.RLock()
	defer i.RUnlock()

	fromBucket := blockTsBucketKey(rangeStart)
	toBucket := blockTsBucketKey(rangeEnd)
	sequentialHeight := getBlockHeight(i.sequentialTopBlock)
	for b := fromBucket; b <= toBucket; b++ {
		blockHeight, exists := i.firstBlockInTsBucket[b]
		if blockHeight > sequentialHeight {
			return 0, false
		} else if exists {
			return blockHeight, true
		}
	}
	return 0, false

}

func (i *blockHeightIndex) validateCandidateBlockHeight(candidateBlockHeight primitives.BlockHeight) (err error) {
	i.RLock()
	defer i.RUnlock()

	topHeight := getBlockHeight(i.topBlock)
	sequentialHeight := getBlockHeight(i.sequentialTopBlock)
	lastWrittenHeight := getBlockHeight(i.lastWrittenBlock)

	if lastWrittenHeight > sequentialHeight && candidateBlockHeight != lastWrittenHeight-1 {
		err = fmt.Errorf("sync session in progress, expected block height %d", lastWrittenHeight-1)

	} else if sequentialHeight == topHeight && candidateBlockHeight <= sequentialHeight {
		err = fmt.Errorf("expected block height higher than current top %d", sequentialHeight)
	}

	if err != nil {
		i.logger.Info(err.Error())
	}
	return
}

func (i *blockHeightIndex) appendBlock(newOffset int64, newBlock *protocol.BlockPairContainer, blockTracker *synchronization.BlockTracker) error {
	newBlockHeight := getBlockHeight(newBlock)
	if err := i.validateCandidateBlockHeight(newBlockHeight); err != nil {
		return err
	}

	i.Lock()
	defer i.Unlock()
	topHeight := getBlockHeight(i.topBlock)
	sequentialHeight := getBlockHeight(i.sequentialTopBlock)
	numTxReceipts := newBlock.ResultsBlock.Header.NumTransactionReceipts()
	blockTs := newBlock.ResultsBlock.Header.Timestamp()

	i.heightOffset[newBlockHeight] = i.nextOffset
	i.nextOffset = newOffset
	// update indices
	i.lastWrittenBlock = newBlock
	lastWrittenHeight := newBlockHeight
	if newBlockHeight > topHeight {
		i.topBlock = newBlock
		topHeight = newBlockHeight
	}

	if lastWrittenHeight == sequentialHeight+1 {
		for height := sequentialHeight + 1; height <= topHeight; height++ {
			if _, ok := i.heightOffset[height]; !ok { // block does not exists
				i.lastWrittenBlock = i.topBlock
				return fmt.Errorf("offset missing for blockHeight (%d), in range (%d - %d) assumed to exist in file storage", uint64(height), uint64(sequentialHeight+1), uint64(topHeight))
			}
			if blockTracker != nil {
				blockTracker.IncrementTo(height)
			}
		}
		i.lastWrittenBlock = i.topBlock
		i.sequentialTopBlock = i.topBlock
	}

	if numTxReceipts > 0 {
		bucketKey := blockTsBucketKey(blockTs)
		firstBlockHeightInBucket, exists := i.firstBlockInTsBucket[bucketKey]
		if !exists || newBlockHeight < firstBlockHeightInBucket {
			i.firstBlockInTsBucket[bucketKey] = newBlockHeight
		}
	}

	return nil
}

func (i *blockHeightIndex) getLastBlock() *protocol.BlockPairContainer {
	i.RLock()
	defer i.RUnlock()
	return i.sequentialTopBlock
}

func (i *blockHeightIndex) getLastBlockHeight() primitives.BlockHeight {
	i.RLock()
	defer i.RUnlock()
	return getBlockHeight(i.sequentialTopBlock)
}

const minuteToNanoRatio = 60 * 1000 * 1000 * 1000

func blockTsBucketKey(nano primitives.TimestampNano) uint32 {
	return uint32(nano / minuteToNanoRatio)
}

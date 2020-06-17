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
	inOrderBlock         *protocol.BlockPairContainer
	topBlock             *protocol.BlockPairContainer
	lastSyncedHeight     primitives.BlockHeight
	logger               log.Logger
}

func newBlockHeightIndex(logger log.Logger, firstBlockOffset int64) *blockHeightIndex {
	return &blockHeightIndex{
		logger:               logger,
		heightOffset:         map[primitives.BlockHeight]int64{},
		firstBlockInTsBucket: map[uint32]primitives.BlockHeight{},
		nextOffset:           firstBlockOffset,
		inOrderBlock:         nil,
		topBlock:             nil,
		lastSyncedHeight:     0,
	}
}

func (i *blockHeightIndex) getSyncState() internodesync.SyncState {
	i.RLock()
	defer i.RUnlock()
	return internodesync.SyncState{
		TopHeight:        getBlockHeight(i.topBlock),
		InOrderHeight:    getBlockHeight(i.inOrderBlock),
		LastSyncedHeight: i.lastSyncedHeight,
	}
}

func (i *blockHeightIndex) getLastSyncedHeight() primitives.BlockHeight {
	i.RLock()
	defer i.RUnlock()

	return i.lastSyncedHeight
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
	inOrderHeight := getBlockHeight(i.inOrderBlock)
	for b := fromBucket; b <= toBucket; b++ {
		blockHeight, exists := i.firstBlockInTsBucket[b]
		if blockHeight > inOrderHeight {
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
	inOrderHeight := getBlockHeight(i.inOrderBlock)

	if i.lastSyncedHeight > inOrderHeight && candidateBlockHeight != i.lastSyncedHeight-1 {
		err = fmt.Errorf("sync session in progress, expected block height %d", i.lastSyncedHeight-1)

	} else if inOrderHeight == topHeight && candidateBlockHeight <= inOrderHeight {
		err = fmt.Errorf("expected block height higher than current top %d",  inOrderHeight)
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
	inOrderHeight := getBlockHeight(i.inOrderBlock)
	numTxReceipts := newBlock.ResultsBlock.Header.NumTransactionReceipts()
	blockTs := newBlock.ResultsBlock.Header.Timestamp()

	i.heightOffset[newBlockHeight] = i.nextOffset
	i.nextOffset = newOffset
	// update indices
	i.lastSyncedHeight = newBlockHeight
	if newBlockHeight > topHeight {
		i.topBlock = newBlock
		topHeight = newBlockHeight
	}
	if i.lastSyncedHeight == inOrderHeight+1 {
			for height := inOrderHeight + 1; height <= topHeight; height++ {
				if _, ok := i.heightOffset[height]; !ok { // block does not exists
					i.lastSyncedHeight = topHeight
					return fmt.Errorf("offset missing for blockHeight (%d), in range (%d - %d) assumed to exist in file storage", uint64(height), uint64(inOrderHeight+1), uint64(topHeight))
				}
				if blockTracker != nil {
					blockTracker.IncrementTo(height)
				}
			}
		i.lastSyncedHeight = topHeight
		i.inOrderBlock = i.topBlock
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
	return i.inOrderBlock
}

func (i *blockHeightIndex) getLastBlockHeight() primitives.BlockHeight {
	i.RLock()
	defer i.RUnlock()
	return getBlockHeight(i.inOrderBlock)
}

const minuteToNanoRatio = 60 * 1000 * 1000 * 1000

func blockTsBucketKey(nano primitives.TimestampNano) uint32 {
	return uint32(nano / minuteToNanoRatio)
}

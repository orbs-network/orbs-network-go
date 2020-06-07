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
	currentOffset        int64
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
		currentOffset:        firstBlockOffset,
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

func (i *blockHeightIndex) fetchCurrentOffset() int64 {
	i.RLock()
	defer i.RUnlock()

	return i.currentOffset
}

func (i *blockHeightIndex) fetchBlockOffset(height primitives.BlockHeight) (offset int64, ok bool) {
	i.RLock()
	defer i.RUnlock()

	offset, ok = i.heightOffset[height]
	return
	//if offset, ok := i.heightOffset[height]; !ok {
	//	return 0, fmt.Errorf("index missing offset for block height %d", height)
	//} else {
	//	return offset, nil
	//}
}

func (i *blockHeightIndex) getEarliestTxBlockInBucketForTsRange(rangeStart primitives.TimestampNano, rangeEnd primitives.TimestampNano) (primitives.BlockHeight, bool) {
	i.RLock()
	defer i.RUnlock()

	fromBucket := blockTsBucketKey(rangeStart)
	toBucket := blockTsBucketKey(rangeEnd)
	for b := fromBucket; b <= toBucket; b++ {
		result, exists := i.firstBlockInTsBucket[b]
		if exists {
			return result, true
		}
	}
	return 0, false

}

func (i *blockHeightIndex) appendBlock(newOffset int64, newBlock *protocol.BlockPairContainer, blockTracker *synchronization.BlockTracker) error {
	i.Lock()
	defer i.Unlock()

	newBlockHeight := getBlockHeight(newBlock)
	topHeight := getBlockHeight(i.topBlock)
	inOrderHeight := getBlockHeight(i.inOrderBlock)
	numTxReceipts := newBlock.ResultsBlock.Header.NumTransactionReceipts()
	blockTs := newBlock.ResultsBlock.Header.Timestamp()

	if _, ok := i.heightOffset[newBlockHeight]; ok { // block exists
		return fmt.Errorf("index of blockHeight (%d) already exists ", uint64(newBlockHeight))
	}

	i.heightOffset[newBlockHeight] = i.currentOffset
	i.currentOffset = newOffset
	// update indices
	i.lastSyncedHeight = newBlockHeight
	if newBlockHeight > topHeight {
		i.topBlock = newBlock
		topHeight = newBlockHeight
	}
	if i.lastSyncedHeight == inOrderHeight+1 {
		if blockTracker != nil {
			for height := inOrderHeight + 1; height <= topHeight; height++ {
				if _, ok := i.heightOffset[height]; !ok { // block does not exists
					panic("block offset not found - should not happen")
				}
				blockTracker.IncrementTo(height)
			}
		}
		i.lastSyncedHeight = topHeight
		i.inOrderBlock = i.topBlock
	}

	if numTxReceipts > 0 {
		_, exists := i.firstBlockInTsBucket[blockTsBucketKey(blockTs)]
		if !exists {
			i.firstBlockInTsBucket[blockTsBucketKey(blockTs)] = newBlockHeight
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

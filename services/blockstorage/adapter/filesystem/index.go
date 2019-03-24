// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package filesystem

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"sync"
)

type blockHeightIndex struct {
	sync.RWMutex
	heightOffset         map[primitives.BlockHeight]int64
	firstBlockInTsBucket map[uint32]primitives.BlockHeight
	topBlock             *protocol.BlockPairContainer
	topBlockHeight       primitives.BlockHeight
	logger               log.BasicLogger
}

func newBlockHeightIndex(logger log.BasicLogger, firstBlockOffset int64) *blockHeightIndex {
	return &blockHeightIndex{
		logger:               logger,
		heightOffset:         map[primitives.BlockHeight]int64{1: firstBlockOffset},
		firstBlockInTsBucket: map[uint32]primitives.BlockHeight{},
		topBlock:             nil,
		topBlockHeight:       0,
	}
}

func (i *blockHeightIndex) fetchTopOffset() int64 {
	i.RLock()
	defer i.RUnlock()

	offset, ok := i.heightOffset[i.topBlockHeight+1]
	if !ok {
		panic(fmt.Sprintf("index missing offset for block height %d", i.topBlockHeight))
	}
	return offset
}

func (i *blockHeightIndex) fetchBlockOffset(height primitives.BlockHeight) int64 {
	i.RLock()
	defer i.RUnlock()

	offset, ok := i.heightOffset[height]
	if !ok {
		panic(fmt.Sprintf("index missing offset for block height %d", height))
	}
	return offset
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

func (i *blockHeightIndex) appendBlock(prevTopOffset int64, newTopOffset int64, newBlock *protocol.BlockPairContainer) error {
	i.Lock()
	defer i.Unlock()

	newBlockHeight := newBlock.ResultsBlock.Header.BlockHeight()
	numTxReceipts := newBlock.ResultsBlock.Header.NumTransactionReceipts()
	blockTs := newBlock.ResultsBlock.Header.Timestamp()

	currentTopOffset, ok := i.heightOffset[i.topBlockHeight+1]
	if !ok {
		return fmt.Errorf("index missing offset for block height %d", i.topBlockHeight)
	}
	if currentTopOffset != prevTopOffset {
		return fmt.Errorf("unexpected top block offest, may be a result of two processes writing concurrently. found offest %d while expecting %d", currentTopOffset, prevTopOffset)
	}

	// update index
	i.topBlock = newBlock
	i.topBlockHeight = newBlock.ResultsBlock.Header.BlockHeight()
	i.heightOffset[newBlockHeight+1] = newTopOffset

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
	return i.topBlock
}

func (i *blockHeightIndex) getLastBlockHeight() primitives.BlockHeight {
	i.RLock()
	defer i.RUnlock()
	return i.topBlockHeight
}

const minuteToNanoRatio = 60 * 1000 * 1000 * 1000

func blockTsBucketKey(nano primitives.TimestampNano) uint32 {
	return uint32(nano / minuteToNanoRatio)
}

package adapter

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"io"
	"os"
	"sync"
)

type blockHeightIndex struct {
	sync.RWMutex
	heightOffset         map[primitives.BlockHeight]int64
	firstBlockInTsBucket map[uint32]primitives.BlockHeight
	topBlock             *protocol.BlockPairContainer
	topBlockHeight       primitives.BlockHeight
}

func newBlockHeightIndex(conf config.FilesystemBlockPersistenceConfig, logger log.BasicLogger) (*blockHeightIndex, error) {
	i := &blockHeightIndex{}
	i.reset()

	file, err := os.Open(blocksFileName(conf))
	if err != nil {
		if os.IsNotExist(err) { // if file does not exist the index is already complete
			return i, nil
		}
		return nil, errors.Wrap(err, "failed to open blocks file for reading")
	}
	defer closeSilently(file, logger)

	err = i.rebuildIndex(file)
	if err != nil {
		return nil, err
	}

	return i, nil
}

func (i *blockHeightIndex) reset() {
	i.Lock()
	defer i.Unlock()

	i.heightOffset = map[primitives.BlockHeight]int64{1: 0}
	i.firstBlockInTsBucket = map[uint32]primitives.BlockHeight{}
	i.topBlockHeight = 0
	i.topBlock = nil
}

func (i *blockHeightIndex) rebuildIndex(r io.Reader) error {
	i.reset()
	offset := int64(0)

	for {
		aBlock, blockSize, err := decode(r)
		if err != nil {
			return nil
		}
		err = i.appendBlock(offset, offset+int64(blockSize), aBlock)
		if err != nil {
			return err
		}
		offset = offset + int64(blockSize)
	}
}

func (i *blockHeightIndex) fetchTopOffest() (int64, error) {
	i.RLock()
	defer i.RUnlock()

	offset, ok := i.heightOffset[i.topBlockHeight]
	if !ok {
		return 0, fmt.Errorf("index missing offset for block height %d", i.topBlockHeight)
	}
	return offset, nil
}

func (i *blockHeightIndex) fetchBlockOffset(height primitives.BlockHeight) (int64, error) {
	i.RLock()
	defer i.RUnlock()

	offset, ok := i.heightOffset[height]
	if !ok {
		return 0, fmt.Errorf("index missing offset for block height %d", height)
	}
	return offset, nil
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

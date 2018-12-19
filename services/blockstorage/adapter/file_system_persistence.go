package adapter

import (
	"bytes"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"io"
	"os"
	"sync"
)

type metrics struct {
	size *metric.Gauge
}

func NewFilesystemBlockPersistence(dataDir string, parent log.BasicLogger, metricFactory metric.Factory) BlockPersistence {
	logger := parent.WithTags(log.String("adapter", "block-storage"))
	return &FilesystemBlockPersistence{
		bhIndex: &blockHeightIndex{
			topHeight:            0,
			heightOffset:         map[primitives.BlockHeight]int64{1: 0},
			firstBlockInTsBucket: map[uint32]primitives.BlockHeight{},
		},
		dataDir:      dataDir,
		metrics:      newMetrics(metricFactory),
		blockTracker: synchronization.NewBlockTracker(logger, 0, 5),
		logger:       logger,
	}
}

type FilesystemBlockPersistence struct {
	bhIndex      *blockHeightIndex
	dataDir      string
	metrics      *metrics
	writeLock    sync.Mutex
	blockTracker *synchronization.BlockTracker
	logger       log.BasicLogger
}

func newMetrics(m metric.Factory) *metrics {
	return &metrics{
		size: m.NewGauge("BlockStorage.InMemoryBlockPersistence.SizeInMB"),
	}
}
func (f *FilesystemBlockPersistence) WriteNextBlock(blockPair *protocol.BlockPairContainer) error {
	f.writeLock.Lock()
	defer f.writeLock.Unlock()

	bh := blockPair.ResultsBlock.Header.BlockHeight()

	currentTop := f.bhIndex.fetchTopHeight()
	if bh != currentTop+1 {
		return fmt.Errorf("attempt to write block %d out of order. current top height is %d", bh, currentTop)
	}

	startOffset, err := f.bhIndex.fetchBlockOffset(bh)
	if err != nil {
		return errors.Wrap(err, "failed to fetch top block offset")
	}

	file, err := os.OpenFile(f.blockFileName(), os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return errors.Wrap(err, "failed to open blocks file for writing")
	}
	defer file.Close()

	currentOffset, err := file.Seek(startOffset, io.SeekStart)
	if err != nil {
		return errors.Wrap(err, "failed to open blocks file for writing")
	}
	if startOffset != currentOffset {
		return errors.Wrapf(err, "failed to seek in blocks file to position %v", startOffset)
	}

	err = encode(blockPair, file)

	if err != nil {
		return errors.Wrap(err, "failed to write block")
	}

	err = file.Sync()
	if err != nil {
		return errors.Wrap(err, "failed to flush blocks file to disk")
	}

	// find our current offset
	currentOffset, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		return errors.Wrap(err, "failed to update block height index")
	}

	err = f.bhIndex.appendBlock(startOffset, currentOffset, blockPair.ResultsBlock.Header.NumTransactionReceipts(), blockPair.ResultsBlock.Header.Timestamp())
	if err != nil {
		return errors.Wrap(err, "failed to update index after writing block")
	}

	f.blockTracker.IncrementHeight()
	return nil
}

func (f *FilesystemBlockPersistence) ScanBlocks(from primitives.BlockHeight, pageSize uint8, cursor CursorFunc) error {
	offset, err := f.bhIndex.fetchBlockOffset(from)
	if err != nil {
		return errors.Wrap(err, "failed to fetch last block")
	}

	file, err := os.Open(f.blockFileName())
	if err != nil {
		return errors.Wrap(err, "failed to open blocks file for reading")
	}
	defer file.Close()

	newOffset, err := file.Seek(offset, io.SeekStart)
	if newOffset != offset || err != nil {
		return errors.Wrapf(err, "failed to seek in blocks file to position %v", offset)
	}

	wantNext := true
	lastHeightRead := primitives.BlockHeight(0)

	for top := f.bhIndex.fetchTopHeight(); wantNext && top > lastHeightRead; {
		currentPage := make([]*protocol.BlockPairContainer, 0, pageSize)
		for ; uint8(len(currentPage)) < pageSize && top > lastHeightRead; top = f.bhIndex.fetchTopHeight() {
			aBlock, err := decode(file)
			if err != nil {
				return errors.Wrapf(err, "failed to decode block")
			}
			currentPage = append(currentPage, aBlock)
			lastHeightRead = aBlock.ResultsBlock.Header.BlockHeight()
		}
		if len(currentPage) > 0 {
			wantNext = cursor(currentPage[0].ResultsBlock.Header.BlockHeight(), currentPage)
		}
	}

	return nil
}

func (f *FilesystemBlockPersistence) GetLastBlockHeight() (primitives.BlockHeight, error) {
	return f.bhIndex.topHeight, nil
}

func (f *FilesystemBlockPersistence) GetLastBlock() (*protocol.BlockPairContainer, error) {
	offset, err := f.bhIndex.fetchTopOffest()
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch last block")
	}

	file, err := os.Open(f.blockFileName())
	if err != nil {
		return nil, errors.Wrap(err, "failed to open blocks file for reading")
	}
	defer file.Close()

	newOffset, err := file.Seek(offset, io.SeekStart)
	if newOffset != offset || err != nil {
		return nil, errors.Wrapf(err, "failed to seek in blocks file to position %v", offset)
	}

	result, err := decode(file)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode block")
	}
	return result, nil
}

func (f *FilesystemBlockPersistence) GetTransactionsBlock(height primitives.BlockHeight) (*protocol.TransactionsBlockContainer, error) {
	bpc, err := f.getBlockAtHeight(height)
	if err != nil {
		return nil, err
	}
	return bpc.TransactionsBlock, nil
}

func (f *FilesystemBlockPersistence) GetResultsBlock(height primitives.BlockHeight) (*protocol.ResultsBlockContainer, error) {
	bpc, err := f.getBlockAtHeight(height)
	if err != nil {
		return nil, err
	}
	return bpc.ResultsBlock, nil
}

func (f *FilesystemBlockPersistence) getBlockAtHeight(height primitives.BlockHeight) (*protocol.BlockPairContainer, error) {
	var bpc *protocol.BlockPairContainer
	err := f.ScanBlocks(height, 1, func(h primitives.BlockHeight, page []*protocol.BlockPairContainer) (wantsMore bool) {
		bpc = page[0]
		return false
	})
	return bpc, err
}

func (f *FilesystemBlockPersistence) GetBlockByTx(txHash primitives.Sha256, minBlockTs primitives.TimestampNano, maxBlockTs primitives.TimestampNano) (block *protocol.BlockPairContainer, txIndexInBlock int, err error) {
	scanFrom, ok := f.bhIndex.getEarliestTxBlockInBucketForTsRange(minBlockTs, maxBlockTs)
	if !ok {
		return nil, 0, nil
	}

	err = f.ScanBlocks(scanFrom, 1, func(h primitives.BlockHeight, page []*protocol.BlockPairContainer) (wantsMore bool) {
		b := page[0]
		if b.ResultsBlock.Header.Timestamp() > maxBlockTs {
			return false
		}
		if b.ResultsBlock.Header.Timestamp() < minBlockTs {
			return true
		}

		for i, receipt := range b.ResultsBlock.TransactionReceipts {
			if bytes.Equal(receipt.Txhash(), txHash) { // found requested transaction
				block = b
				txIndexInBlock = i
				return false
			}
		}
		return true
	})

	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to fetch block by txHash")
	}
	return block, txIndexInBlock, nil
}

func (f *FilesystemBlockPersistence) GetBlockTracker() *synchronization.BlockTracker {
	return f.blockTracker
}

func (f *FilesystemBlockPersistence) blockFileName() string {
	return f.dataDir + "/blocks"
}

type blockHeightIndex struct {
	sync.RWMutex
	topHeight            primitives.BlockHeight
	heightOffset         map[primitives.BlockHeight]int64
	firstBlockInTsBucket map[uint32]primitives.BlockHeight
}

func (i *blockHeightIndex) fetchTopOffest() (int64, error) {
	i.RLock()
	defer i.RUnlock()

	topHeight := i.topHeight
	offset, ok := i.heightOffset[topHeight]
	if !ok {
		return 0, fmt.Errorf("index missing offset for block height %d", topHeight)
	}
	return offset, nil
}

func (i *blockHeightIndex) fetchTopHeight() (height primitives.BlockHeight) {
	i.RLock()
	defer i.RUnlock()

	return i.topHeight
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

func (i *blockHeightIndex) appendBlock(prevTopOffset int64, newTopOffset int64, numTxReceipts uint32, blockTs primitives.TimestampNano) error {
	i.Lock()
	defer i.Unlock()

	currentTopOffset, ok := i.heightOffset[i.topHeight+1]
	if !ok {
		return fmt.Errorf("index missing offset for block height %d", i.topHeight)
	}
	if currentTopOffset != prevTopOffset {
		return fmt.Errorf("unexpected top block offest, may be a result of two processes writing concurrently. found offest %d while expecting %d", currentTopOffset, prevTopOffset)
	}
	i.topHeight++
	i.heightOffset[i.topHeight+1] = newTopOffset

	if numTxReceipts > 0 {
		_, exists := i.firstBlockInTsBucket[blockTsBucketKey(blockTs)]
		if !exists {
			i.firstBlockInTsBucket[blockTsBucketKey(blockTs)] = i.topHeight
		}
	}

	return nil
}

const minuteToNanoRatio = 60 * 1000 * 1000 * 1000

func blockTsBucketKey(nano primitives.TimestampNano) uint32 {
	return uint32(nano / minuteToNanoRatio)
}

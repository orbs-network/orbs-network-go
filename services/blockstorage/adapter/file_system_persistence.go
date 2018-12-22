package adapter

import (
	"bytes"
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type metrics struct {
	size *metric.Gauge
}

func NewFilesystemBlockPersistence(ctx context.Context, c config.FilesystemBlockPersistenceConfig, parent log.BasicLogger, metricFactory metric.Factory) (BlockPersistence, error) {
	logger := parent.WithTags(log.String("adapter", "block-storage"))
	adapter := &FilesystemBlockPersistence{
		bhIndex: newBlockHeightIndex(),
		config:  c,
		metrics: newMetrics(metricFactory),
		logger:  logger,
	}

	err := adapter.refreshIndex()
	if err != nil {
		return nil, err
	}
	adapter.blockTracker = synchronization.NewBlockTracker(logger, uint64(adapter.bhIndex.topBlockHeight), 5)

	newTip, err := newWritingTip(ctx, c.DataDir(), adapter.blockFileName(), logger)
	if err != nil {
		return nil, err
	}

	adapter.tip = newTip
	return adapter, nil
}

func newWritingTip(ctx context.Context, dir, filename string, logger log.BasicLogger) (*writingTip, error) {
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return nil, errors.Wrap(err, "failed to verify data directory exists")
	}
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open blocks file for writing")
	}
	result := &writingTip{
		file: file,
	}

	go func() {
		<-ctx.Done()
		result.Lock()
		defer result.Unlock()
		err := file.Close()
		if err != nil {
			logger.Error("failed to close blocks file", log.String("filename", result.file.Name()))
			return
		}
		logger.Info("closed blocks file", log.String("filename", result.file.Name()))

	}()

	return result, nil
}

type writingTip struct {
	sync.Mutex
	file       *os.File
	currentPos int64
}

func (wh *writingTip) writeBlockAtOffset(pos int64, blockPair *protocol.BlockPairContainer) (int64, error) {
	if pos != wh.currentPos {
		currentOffset, err := wh.file.Seek(pos, io.SeekStart)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to seek writing tip to pos %d", pos)
		}
		if pos != currentOffset {
			return 0, errors.Wrapf(err, "failed to seek in blocks file to position %v", pos)
		}
	}

	err := encode(blockPair, wh.file)
	if err != nil {
		return 0, errors.Wrap(err, "failed to write block")
	}

	err = wh.file.Sync()
	if err != nil {
		return 0, errors.Wrap(err, "failed to flush blocks file to disk")
	}
	// find our current offset
	newPos, err := wh.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, errors.Wrap(err, "failed to update block height index")
	}
	wh.currentPos = newPos // assign only after checking err
	return newPos, nil
}

type FilesystemBlockPersistence struct {
	config       config.FilesystemBlockPersistenceConfig
	bhIndex      *blockHeightIndex
	metrics      *metrics
	blockTracker *synchronization.BlockTracker
	logger       log.BasicLogger
	tip          *writingTip
}

func newMetrics(m metric.Factory) *metrics {
	return &metrics{
		size: m.NewGauge("BlockStorage.FilesystemBlockPersistence.SizeInBytes"),
	}
}

func (f *FilesystemBlockPersistence) WriteNextBlock(blockPair *protocol.BlockPairContainer) error {
	f.tip.Lock()
	defer f.tip.Unlock()

	bh := blockPair.ResultsBlock.Header.BlockHeight()

	currentTop := f.bhIndex.getLastBlockHeight()
	if bh != currentTop+1 {
		return fmt.Errorf("attempt to write block %d out of order. current top height is %d", bh, currentTop)
	}

	startPos, err := f.bhIndex.fetchBlockOffset(bh)
	if err != nil {
		return errors.Wrap(err, "failed to fetch top block offset")
	}

	newPos, err := f.tip.writeBlockAtOffset(startPos, blockPair)
	if err != nil {
		return err
	}

	err = f.bhIndex.appendBlock(startPos, newPos, blockPair)
	if err != nil {
		return errors.Wrap(err, "failed to update index after writing block")
	}

	f.blockTracker.IncrementHeight()
	f.metrics.size.Add(newPos - startPos)

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

	for top := f.bhIndex.getLastBlockHeight(); wantNext && top > lastHeightRead; {
		currentPage := make([]*protocol.BlockPairContainer, 0, pageSize)
		for ; uint8(len(currentPage)) < pageSize && top > lastHeightRead; top = f.bhIndex.getLastBlockHeight() {
			aBlock, _, err := decode(file)
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
	return f.bhIndex.getLastBlockHeight(), nil
}

func (f *FilesystemBlockPersistence) GetLastBlock() (*protocol.BlockPairContainer, error) {
	return f.bhIndex.getLastBlock(), nil
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
	return filepath.Join(f.config.DataDir(), f.config.BlocksFilename())
}

func (f *FilesystemBlockPersistence) refreshIndex() error {
	f.bhIndex.reset()
	offset := int64(0)

	file, err := os.Open(f.blockFileName())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Wrap(err, "failed to open blocks file for reading")
	}
	defer file.Close()

	for {
		aBlock, blockSize, err := decode(file)
		if err != nil {
			return nil
		}
		f.bhIndex.appendBlock(offset, offset+int64(blockSize), aBlock)
		offset = offset + int64(blockSize)
	}

	return nil
}

type blockHeightIndex struct {
	sync.RWMutex
	heightOffset         map[primitives.BlockHeight]int64
	firstBlockInTsBucket map[uint32]primitives.BlockHeight
	topBlock             *protocol.BlockPairContainer
	topBlockHeight       primitives.BlockHeight
}

func newBlockHeightIndex() *blockHeightIndex {
	i := &blockHeightIndex{}
	i.reset()
	return i
}
func (i *blockHeightIndex) reset() {
	i.RLock()
	defer i.RUnlock()

	i.heightOffset = map[primitives.BlockHeight]int64{1: 0}
	i.firstBlockInTsBucket = map[uint32]primitives.BlockHeight{}
	i.topBlockHeight = 0
	i.topBlock = nil
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

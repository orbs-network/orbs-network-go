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
)

type metrics struct {
	size *metric.Gauge
}

const blocksFilename = "blocks"

func newMetrics(m metric.Factory) *metrics {
	return &metrics{
		size: m.NewGauge("BlockStorage.FilesystemBlockPersistence.SizeInBytes"),
	}
}

type blocksCodec interface {
	encode(block *protocol.BlockPairContainer, w io.Writer) error
	decode(r io.Reader) (*protocol.BlockPairContainer, int, error)
}

type FilesystemBlockPersistence struct {
	config       config.FilesystemBlockPersistenceConfig
	bhIndex      *blockHeightIndex
	metrics      *metrics
	blockTracker *synchronization.BlockTracker
	logger       log.BasicLogger
	tip          *writingTip
	codec        blocksCodec
}

func NewFilesystemBlockPersistence(ctx context.Context, conf config.FilesystemBlockPersistenceConfig, parent log.BasicLogger, metricFactory metric.Factory) (BlockPersistence, error) {
	logger := parent.WithTags(log.String("adapter", "block-storage"))

	codec := newSimpleCodec(conf.BlockStorageMaxBlockSize())

	// creates the file if missing
	newTip, err := newWritingTip(ctx, conf.BlockStorageDataDir(), blocksFileName(conf), codec, logger)
	if err != nil {
		return nil, err
	}

	bhIndex, err := constructIndexFromFile(blocksFileName(conf), logger, codec)
	if err != nil {
		return nil, err
	}

	adapter := &FilesystemBlockPersistence{
		bhIndex:      bhIndex,
		config:       conf,
		blockTracker: synchronization.NewBlockTracker(logger, uint64(bhIndex.topBlockHeight), 5),
		metrics:      newMetrics(metricFactory),
		logger:       logger,
		tip:          newTip,
		codec:        codec,
	}

	return adapter, nil
}

func constructIndexFromFile(filename string, logger log.BasicLogger, c blocksCodec) (*blockHeightIndex, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open blocks file for reading")
	}
	defer closeSilently(file, logger)

	return constructIndexFromReader(file, logger, c)
}

func constructIndexFromReader(r io.Reader, logger log.BasicLogger, c blocksCodec) (*blockHeightIndex, error) {
	bhIndex := newBlockHeightIndex()
	offset := int64(0)
	for {
		aBlock, blockSize, err := c.decode(r)
		if err != nil {
			if err == io.EOF {
				logger.Info("built index", log.Int64("valid-block-bytes", offset), log.BlockHeight(bhIndex.topBlockHeight))
			} else {
				logger.Error("built index, found and ignoring invalid block records", log.Int64("valid-block-bytes", offset), log.Error(err), log.BlockHeight(bhIndex.topBlockHeight))
			}
			break // index up to EOF or first invalid record.
		}
		err = bhIndex.appendBlock(offset, offset+int64(blockSize), aBlock)
		if err != nil {
			return nil, errors.Wrap(err, "failed building block height index")
		}
		offset = offset + int64(blockSize)
	}
	return bhIndex, nil
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

	f.blockTracker.IncrementTo(currentTop + 1)
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
	defer closeSilently(file, f.logger)

	newOffset, err := file.Seek(offset, io.SeekStart)
	if newOffset != offset || err != nil {
		return errors.Wrapf(err, "failed to seek in blocks file to position %v", offset)
	}

	wantNext := true
	eof := false

	for wantNext && !eof {
		page := make([]*protocol.BlockPairContainer, 0, pageSize)

		for uint8(len(page)) < pageSize {
			aBlock, _, err := f.codec.decode(file)
			if err != nil {
				if err == io.EOF {
					eof = true
					break
				}
				return errors.Wrapf(err, "failed to decode block")
			}
			page = append(page, aBlock)
		}
		if len(page) > 0 {
			wantNext = cursor(page[0].ResultsBlock.Header.BlockHeight(), page)
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
	return blocksFileName(f.config)
}

func blocksFileName(config config.FilesystemBlockPersistenceConfig) string {
	return filepath.Join(config.BlockStorageDataDir(), blocksFilename)
}

func closeSilently(file *os.File, logger log.BasicLogger) {
	err := file.Close()
	if err != nil {
		logger.Error("failed to close file", log.Error(err), log.String("filename", file.Name()))
	}
}

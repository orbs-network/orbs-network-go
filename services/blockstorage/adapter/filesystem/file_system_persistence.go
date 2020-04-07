// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package filesystem

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

type metrics struct {
	sizeOnDisk *metric.Gauge
}

const blocksFilename = "blocks"

func newMetrics(m metric.Factory) *metrics {
	return &metrics{
		sizeOnDisk: m.NewGauge("BlockStorage.FileSystemSize.Bytes"),
	}
}

type blockCodec interface {
	encode(block *protocol.BlockPairContainer, w io.Writer) (int, error)
	decode(r io.Reader) (*protocol.BlockPairContainer, int, error)
}

type BlockPersistence struct {
	config       config.FilesystemBlockPersistenceConfig
	bhIndex      *blockHeightIndex
	metrics      *metrics
	blockTracker *synchronization.BlockTracker
	logger       log.Logger
	blockWriter  *blockWriter
	codec        blockCodec
}

func (f *BlockPersistence) GracefulShutdown(shutdownContext context.Context) {
	logger := f.logger.WithTags(log.String("filename", blocksFileName(f.config)))
	if err := f.blockWriter.Close(); err != nil {
		logger.Error("failed to close blocks file")
		return
	}
	logger.Info("closed blocks file")
}

func NewBlockPersistence(conf config.FilesystemBlockPersistenceConfig, parent log.Logger, metricFactory metric.Factory) (*BlockPersistence, error) {
	logger := parent.WithTags(log.String("adapter", "block-storage"))

	codec := newCodec(conf.BlockStorageFileSystemMaxBlockSizeInBytes())

	file, blocksOffset, err := openBlocksFile(conf, logger)
	if err != nil {
		return nil, err
	}

	bhIndex, err := buildIndex(bufio.NewReaderSize(file, 1024*1024), blocksOffset, logger, codec)
	if err != nil {
		closeSilently(file, logger)
		return nil, err
	}

	newTip, err := newFileBlockWriter(file, codec, bhIndex.fetchTopOffset())
	if err != nil {
		closeSilently(file, logger)
		return nil, err
	}

	adapter := &BlockPersistence{
		bhIndex:      bhIndex,
		config:       conf,
		blockTracker: synchronization.NewBlockTracker(logger, uint64(bhIndex.topBlockHeight), 5),
		metrics:      newMetrics(metricFactory),
		logger:       logger,
		blockWriter:  newTip,
		codec:        codec,
	}

	if size, err := getBlockFileSize(file); err != nil {
		return adapter, err
	} else {
		adapter.metrics.sizeOnDisk.Add(size)
	}

	return adapter, nil
}

func getBlockFileSize(file *os.File) (int64, error) {
	if fi, err := file.Stat(); err != nil {
		return 0, errors.Wrap(err, "unable to read file size for metrics")
	} else {
		return fi.Size(), nil
	}
}

func openBlocksFile(conf config.FilesystemBlockPersistenceConfig, logger log.Logger) (*os.File, int64, error) {
	dir := conf.BlockStorageFileSystemDataDir()
	filename := blocksFileName(conf)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "failed to verify data directory exists %s", dir)
	}

	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "failed to open blocks file for writing %s", filename)
	}

	err = advisoryLockExclusive(file)
	if err != nil {
		closeSilently(file, logger)
		return nil, 0, errors.Wrapf(err, "failed to obtain exclusive lock for writing %s", filename)
	}

	firstBlockOffset, err := validateFileHeader(file, conf, logger)
	if err != nil {
		closeSilently(file, logger)
		return nil, 0, errors.Wrapf(err, "failed to obtain exclusive lock for writing %s", filename)
	}

	return file, firstBlockOffset, nil
}

func validateFileHeader(file *os.File, conf config.FilesystemBlockPersistenceConfig, logger log.Logger) (int64, error) {

	info, err := file.Stat()
	if err != nil {
		return 0, err
	}
	if info.Size() == 0 { // empty file
		if err := writeNewFileHeader(file, conf, logger); err != nil {
			return 0, err
		}
	}

	offset, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return 0, errors.Wrapf(err, "error reading blocks file header")
	}
	if offset != 0 {
		return 0, fmt.Errorf("error reading blocks file header")
	}

	header := newBlocksFileHeader(0, 0)
	err = header.read(file)
	if err != nil {
		return 0, errors.Wrapf(err, "error reading blocks file header")
	}

	if header.NetworkType != uint32(conf.NetworkType()) {
		return 0, fmt.Errorf("blocks file network type mismatch. found netowrk type %d expected %d", header.NetworkType, conf.NetworkType())
	}

	if header.ChainId != uint32(conf.VirtualChainId()) {
		return 0, fmt.Errorf("blocks file virtual chain id mismatch. found vchain id %d expected %d", header.ChainId, conf.VirtualChainId())
	}

	offset, err = file.Seek(0, io.SeekCurrent) // read current offset
	if err != nil {
		return 0, errors.Wrapf(err, "error reading blocks file header")
	}

	return offset, nil
}

func writeNewFileHeader(file *os.File, conf config.FilesystemBlockPersistenceConfig, logger log.Logger) error {
	header := newBlocksFileHeader(uint32(conf.NetworkType()), uint32(conf.VirtualChainId()))
	logger.Info("creating new blocks file", log.String("filename", file.Name()))
	err := header.write(file)
	if err != nil {
		return errors.Wrapf(err, "error writing blocks file header")
	}
	err = file.Sync()
	if err != nil {
		return errors.Wrapf(err, "error writing blocks file header")
	}
	return nil
}

func advisoryLockExclusive(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}

func newFileBlockWriter(file *os.File, codec blockCodec, nextBlockOffset int64) (*blockWriter, error) {
	newOffset, err := file.Seek(nextBlockOffset, io.SeekStart)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to seek to next block offset %d", nextBlockOffset)
	}
	if newOffset != nextBlockOffset {
		return nil, fmt.Errorf("failed to seek to next block offset. requested offset %d, but new reached %d", nextBlockOffset, newOffset)
	}

	result := newBlockWriter(file, codec)

	return result, nil
}

func buildIndex(r io.Reader, firstBlockOffset int64, logger log.Logger, c blockCodec) (*blockHeightIndex, error) {
	bhIndex := newBlockHeightIndex(logger, firstBlockOffset)
	offset := int64(firstBlockOffset)
	for {
		aBlock, blockSize, err := c.decode(r)
		if err != nil {
			if err == io.EOF {
				logger.Info("built index", log.Int64("valid-block-bytes", offset), logfields.BlockHeight(bhIndex.topBlockHeight))
			} else {
				logger.Error("built index, found and ignoring invalid block records", log.Int64("valid-block-bytes", offset), log.Error(err), logfields.BlockHeight(bhIndex.topBlockHeight))
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

func (f *BlockPersistence) WriteNextBlock(blockPair *protocol.BlockPairContainer) (bool, primitives.BlockHeight, error) {
	f.blockWriter.Lock()
	defer f.blockWriter.Unlock()

	bh := blockPair.ResultsBlock.Header.BlockHeight()

	currentTop := f.bhIndex.getLastBlockHeight()
	if bh != currentTop+1 {
		return false, currentTop, nil
	}

	n, err := f.blockWriter.writeBlock(blockPair)
	if err != nil {
		return false, currentTop, err
	}

	startPos := f.bhIndex.fetchBlockOffset(bh)
	err = f.bhIndex.appendBlock(startPos, startPos+int64(n), blockPair)
	if err != nil {
		return false, currentTop, errors.Wrap(err, "failed to update index after writing block")
	}

	f.blockTracker.IncrementTo(bh)
	f.metrics.sizeOnDisk.Add(int64(n))

	return true, bh, nil
}

func (f *BlockPersistence) ScanBlocks(from primitives.BlockHeight, pageSize uint64, cursor adapter.CursorFunc) error {
	currentTop := f.bhIndex.topBlockHeight
	if currentTop < from {
		return fmt.Errorf("requested unknown block height %d. current height is %d", from, currentTop)
	}

	file, err := os.Open(f.blockFileName())
	if err != nil {
		return errors.Wrap(err, "failed to open blocks file for reading")
	}
	defer closeSilently(file, f.logger)

	initialOffset := f.bhIndex.fetchBlockOffset(from)
	newOffset, err := file.Seek(initialOffset, io.SeekStart)
	if newOffset != initialOffset || err != nil {
		return errors.Wrapf(err, "failed to seek in blocks file to position %v", initialOffset)
	}

	wantNext := true
	eof := false

	for wantNext && !eof {
		page := make([]*protocol.BlockPairContainer, 0, pageSize)

		for uint64(len(page)) < pageSize {
			aBlock, _, err := f.codec.decode(file)
			if err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
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

func (f *BlockPersistence) GetLastBlockHeight() (primitives.BlockHeight, error) {
	return f.bhIndex.getLastBlockHeight(), nil
}

func (f *BlockPersistence) GetLastBlock() (*protocol.BlockPairContainer, error) {
	return f.bhIndex.getLastBlock(), nil
}

func (f *BlockPersistence) GetTransactionsBlock(height primitives.BlockHeight) (*protocol.TransactionsBlockContainer, error) {
	bpc, err := f.getBlockAtHeight(height)
	if err != nil {
		return nil, err
	}
	return bpc.TransactionsBlock, nil
}

func (f *BlockPersistence) GetResultsBlock(height primitives.BlockHeight) (*protocol.ResultsBlockContainer, error) {
	bpc, err := f.getBlockAtHeight(height)
	if err != nil {
		return nil, err
	}
	return bpc.ResultsBlock, nil
}

func (f *BlockPersistence) getBlockAtHeight(height primitives.BlockHeight) (*protocol.BlockPairContainer, error) {
	var bpc *protocol.BlockPairContainer
	err := f.ScanBlocks(height, 1, func(h primitives.BlockHeight, page []*protocol.BlockPairContainer) (wantsMore bool) {
		bpc = page[0]
		return false
	})
	return bpc, err
}

func (f *BlockPersistence) GetBlockByTx(txHash primitives.Sha256, minBlockTs primitives.TimestampNano, maxBlockTs primitives.TimestampNano) (block *protocol.BlockPairContainer, txIndexInBlock int, err error) {
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

func (f *BlockPersistence) GetBlockTracker() *synchronization.BlockTracker {
	return f.blockTracker
}

func (f *BlockPersistence) blockFileName() string {
	return blocksFileName(f.config)
}

func blocksFileName(config config.FilesystemBlockPersistenceConfig) string {
	return filepath.Join(config.BlockStorageFileSystemDataDir(), blocksFilename)
}

func closeSilently(file *os.File, logger log.Logger) {
	err := file.Close()
	if err != nil {
		logger.Error("failed to close file", log.Error(err), log.String("filename", file.Name()))
	}
}

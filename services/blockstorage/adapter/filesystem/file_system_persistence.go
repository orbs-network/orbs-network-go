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
	"github.com/orbs-network/orbs-network-go/services/blockstorage/internodesync"
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

func (f *BlockPersistence) GetSyncState() internodesync.SyncState {
	return f.bhIndex.getSyncState()
}

func getBlockHeight(block *protocol.BlockPairContainer) primitives.BlockHeight {
	if block == nil {
		return 0
	}
	return block.TransactionsBlock.Header.BlockHeight()
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

	newTip, err := newFileBlockWriter(file, codec, bhIndex.fetchNextOffset())
	if err != nil {
		closeSilently(file, logger)
		return nil, err
	}

	adapter := &BlockPersistence{
		bhIndex:      bhIndex,
		config:       conf,
		blockTracker: synchronization.NewBlockTracker(logger, uint64(bhIndex.getLastBlockHeight()), 5),
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
				logger.Info("built index", log.Int64("valid-block-bytes", offset), logfields.BlockHeight(bhIndex.getLastBlockHeight()))
			} else {
				logger.Error("built index, found and ignoring invalid block records", log.Int64("valid-block-bytes", offset), log.Error(err), logfields.BlockHeight(bhIndex.getLastBlockHeight()))
			}
			break // index up to EOF or first invalid record.
		}
		err = bhIndex.appendBlock(offset+int64(blockSize), aBlock, nil)
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

	bh := getBlockHeight(blockPair)

	syncState := f.bhIndex.getSyncState()
	if err := f.bhIndex.validateCandidateBlockHeight(bh); err != nil {
		return false, syncState.InOrderHeight, nil
	}

	n, err := f.blockWriter.writeBlock(blockPair)
	if err != nil {
		return false, syncState.InOrderHeight, err
	}

	startPos := f.bhIndex.fetchNextOffset()
	err = f.bhIndex.appendBlock(startPos+int64(n), blockPair, f.blockTracker)
	if err != nil {
		return false, syncState.InOrderHeight, errors.Wrap(err, "failed to update index after writing block")
	}

	f.metrics.sizeOnDisk.Add(int64(n))
	return true, f.bhIndex.getLastBlockHeight(), nil
}

func (f *BlockPersistence) ScanBlocks(from primitives.BlockHeight, pageSize uint8, cursor adapter.CursorFunc) error {

	inOrderHeight := f.bhIndex.getLastBlockHeight()
	if (inOrderHeight < from) || from == 0 {
		return fmt.Errorf("requested unsupported block height %d. Supported range for scan is determined by inOrder(%d)", from, inOrderHeight)
	}

	file, err := os.Open(f.blockFileName())
	if err != nil {
		return errors.Wrap(err, "failed to open blocks file for reading")
	}
	defer closeSilently(file, f.logger)

	fromHeight := from
	wantsMore := true
	eof := false
	for fromHeight <= inOrderHeight && wantsMore && !eof {
		toHeight := fromHeight + primitives.BlockHeight(pageSize) - 1
		if toHeight > inOrderHeight {
			toHeight = inOrderHeight
		}
		page := make([]*protocol.BlockPairContainer, 0, pageSize)
		// TODO: Gad allow update of inOrder inside page
		for height := fromHeight; height <= toHeight; height++ {
			aBlock, err := f.fetchBlockFromFile(height, file)
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
			wantsMore = cursor(page[0].ResultsBlock.Header.BlockHeight(), page)
		}
		inOrderHeight = f.bhIndex.getLastBlockHeight()
		fromHeight = toHeight + 1
	}

	return nil
}

func (f *BlockPersistence) fetchBlockFromFile(height primitives.BlockHeight, file *os.File) (*protocol.BlockPairContainer, error) {
	initialOffset, ok := f.bhIndex.fetchBlockOffset(height)
	if !ok {
		return nil, fmt.Errorf("failed to find requested block %d", uint64(height))
	}
	newOffset, err := file.Seek(initialOffset, io.SeekStart)
	if newOffset != initialOffset || err != nil {
		return nil, errors.Wrapf(err, "failed to seek in blocks file to position %v", initialOffset)
	}
	aBlock, _, err := f.codec.decode(file)
	return aBlock, err
}

func (f *BlockPersistence) GetLastBlockHeight() (primitives.BlockHeight, error) {
	return f.bhIndex.getLastBlockHeight(), nil
}

func (f *BlockPersistence) GetLastBlock() (*protocol.BlockPairContainer, error) {
	return f.bhIndex.getLastBlock(), nil
}

func (f *BlockPersistence) GetTransactionsBlock(height primitives.BlockHeight) (*protocol.TransactionsBlockContainer, error) {
	bpc, err := f.GetBlock(height)
	if err != nil {
		return nil, err
	}
	return bpc.TransactionsBlock, nil
}

func (f *BlockPersistence) GetResultsBlock(height primitives.BlockHeight) (*protocol.ResultsBlockContainer, error) {
	bpc, err := f.GetBlock(height)
	if err != nil {
		return nil, err
	}
	return bpc.ResultsBlock, nil
}

func (f *BlockPersistence) GetBlock(height primitives.BlockHeight) (*protocol.BlockPairContainer, error) {
	file, err := os.Open(f.blockFileName())
	if err != nil {
		return nil, errors.Wrap(err, "failed to open blocks file for reading")
	}
	defer closeSilently(file, f.logger)

	if aBlock, err := f.fetchBlockFromFile(height, file); err != nil {
		return nil, errors.Wrapf(err, "failed to decode block")
	} else {
		return aBlock, nil
	}
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

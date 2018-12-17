package adapter

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"io"
	"os"
	"sync"
)

type dataFile struct {
	sync.RWMutex
	dataDir      string
	topHeight    primitives.BlockHeight
	heightOffset map[primitives.BlockHeight]int64
}

func NewFilesystemBlockPersistence(dataDir string) BlockPersistence {
	return &FilesystemBlockPersistence{
		dataFile: dataFile{
			dataDir:      dataDir,
			topHeight:    0,
			heightOffset: map[primitives.BlockHeight]int64{1: 0},
		},
	}
}

type FilesystemBlockPersistence struct {
	dataFile dataFile
}

// TODO - make sure we open files with appropriate locking to prevent concurrent collisions

func (f *FilesystemBlockPersistence) WriteNextBlock(blockPair *protocol.BlockPairContainer) error {
	f.dataFile.Lock()
	defer f.dataFile.Unlock()

	bh := blockPair.ResultsBlock.Header.BlockHeight()
	if bh != f.dataFile.topHeight+1 {
		return fmt.Errorf("attempt to write block %d out of order. current top height is %d", bh, f.dataFile.topHeight)
	}

	offset, ok := f.dataFile.heightOffset[bh]
	if !ok {
		return fmt.Errorf("index missing offset for block height %d", bh)
	}

	file, err := os.OpenFile(f.blockFileName(), os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return errors.Wrap(err, "failed to open blocks file for writing")
	}
	newOffset, err := file.Seek(offset, io.SeekStart)
	if err != nil {
		return errors.Wrap(err, "failed to open blocks file for writing")
	}
	if offset != newOffset {
		return errors.Wrapf(err, "failed to seek in blocks file to position %v", offset)
	}

	err = encode(blockPair, file)

	if err != nil {
		return errors.Wrap(err, "failed to write block")
	}

	err = file.Sync()
	if err != nil {
		return errors.Wrap(err, "failed to flush blocks file to disk")
	}

	newOffset, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		return errors.Wrap(err, "failed to update block height index")
	}

	f.dataFile.topHeight++
	f.dataFile.heightOffset[bh+1] = newOffset
	return nil
}

// TODO - don't lock mutex for thew entire scan duration
// TODO - make sure we open files with appropriate locking to prevent concurrent collisions
func (f *FilesystemBlockPersistence) ScanBlocks(from primitives.BlockHeight, pageSize uint8, cursor CursorFunc) error {
	f.dataFile.RLock()
	defer f.dataFile.RUnlock()

	offset, ok := f.dataFile.heightOffset[from]
	if !ok {
		return fmt.Errorf("index missing offset for block height %d", f.dataFile.topHeight)
	}

	file, err := os.Open(f.blockFileName())
	if err != nil {
		return errors.Wrap(err, "failed to open blocks file for reading")
	}

	newOffset, err := file.Seek(offset, io.SeekStart)
	if newOffset != offset || err != nil {
		return errors.Wrapf(err, "failed to seek in blocks file to position %v", offset)
	}

	wantNext := true
	lastHeight := primitives.BlockHeight(0)

	// TODO - check the index under a short lock at any iteration
	for wantNext && f.dataFile.topHeight > lastHeight {
		currentPage := make([]*protocol.BlockPairContainer, 0, pageSize)
		for uint8(len(currentPage)) < pageSize && f.dataFile.topHeight > lastHeight {
			aBlock, err := decode(file)
			if err != nil {
				return errors.Wrapf(err, "failed to decode block")
			}
			currentPage = append(currentPage, aBlock)
		}
		if len(currentPage) > 0 {
			wantNext = cursor(currentPage[0].ResultsBlock.Header.BlockHeight(), currentPage)
		}
	}

	return nil
}

func (*FilesystemBlockPersistence) GetLastBlockHeight() (primitives.BlockHeight, error) {
	panic("implement me")
}

// TODO - don't lock mutex for thew entire scan duration
// TODO - make sure we open files with appropriate locking to prevent concurrent collisions
func (f *FilesystemBlockPersistence) GetLastBlock() (*protocol.BlockPairContainer, error) {
	f.dataFile.RLock()
	defer f.dataFile.RUnlock()

	offset, ok := f.dataFile.heightOffset[f.dataFile.topHeight]
	if !ok {
		return nil, fmt.Errorf("index missing offset for block height %d", f.dataFile.topHeight)
	}

	file, err := os.Open(f.blockFileName())
	if err != nil {
		return nil, errors.Wrap(err, "failed to open blocks file for reading")
	}

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

func (*FilesystemBlockPersistence) GetTransactionsBlock(height primitives.BlockHeight) (*protocol.TransactionsBlockContainer, error) {
	panic("implement me")
}

func (*FilesystemBlockPersistence) GetResultsBlock(height primitives.BlockHeight) (*protocol.ResultsBlockContainer, error) {
	panic("implement me")
}

func (*FilesystemBlockPersistence) GetBlockByTx(txHash primitives.Sha256, minBlockTs primitives.TimestampNano, maxBlockTs primitives.TimestampNano) (block *protocol.BlockPairContainer, txIndexInBlock int, err error) {
	panic("implement me")
}

func (*FilesystemBlockPersistence) GetBlockTracker() *synchronization.BlockTracker {
	panic("implement me")
}

func (f *FilesystemBlockPersistence) blockFileName() string {
	return f.dataFile.dataDir + "/blocks"
}

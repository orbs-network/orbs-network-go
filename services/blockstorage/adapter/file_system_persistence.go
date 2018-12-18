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

func NewFilesystemBlockPersistence(dataDir string) BlockPersistence {
	return &FilesystemBlockPersistence{
		bhIndex: &blockHeightIndex{
			topHeight:    0,
			heightOffset: map[primitives.BlockHeight]int64{1: 0},
		},
		dataDir: dataDir,
	}
}

type FilesystemBlockPersistence struct {
	bhIndex *blockHeightIndex
	dataDir string

	writeLock sync.Mutex
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

	err = f.bhIndex.appendBlock(startOffset, currentOffset)
	if err != nil {
		return errors.Wrap(err, "failed to update index after writing block")
	}
	return nil
}

// TODO - make sure we open files with appropriate locking to prevent concurrent collisions

// TODO - don't lock mutex for thew entire scan duration
// TODO - make sure we open files with appropriate locking to prevent concurrent collisions
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

// TODO - make sure we open files with appropriate locking to prevent concurrent collisions
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
	return f.dataDir + "/blocks"
}

type blockHeightIndex struct {
	sync.RWMutex
	topHeight    primitives.BlockHeight
	heightOffset map[primitives.BlockHeight]int64
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

func (i *blockHeightIndex) appendBlock(prevTopOffset int64, newTopOffset int64) error {
	i.Lock()
	defer i.Unlock()

	currentTopOffest, ok := i.heightOffset[i.topHeight+1]
	if !ok {
		return fmt.Errorf("index missing offset for block height %d", i.topHeight)
	}
	if currentTopOffest != prevTopOffset {
		return fmt.Errorf("unexpected top block offest, may be a result of two processes writing concurrently. found offest %d while expecting %d", currentTopOffest, prevTopOffset)
	}
	i.topHeight++
	i.heightOffset[i.topHeight+1] = newTopOffset
	return nil
}

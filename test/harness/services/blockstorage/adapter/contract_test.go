package adapter

import (
	"encoding/binary"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/gossip/codec"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"sync"
	"testing"
)

func TestBlockPersistenceContract_WritesBlockAndRetrieves(t *testing.T) {
	t.Run("In Memory Adapter", testWritesBlockAndRetrieves(anInMemoryAdapter))
	t.Run("Filesystem Adapter", testWritesBlockAndRetrieves(aFileSystemAdapter))
}

func TestBlockPersistenceContract_FailToWriteOutOfOrder(t *testing.T) {
	t.Run("In Memory Adapter", testFailToWriteOutOfOrder(anInMemoryAdapter))
	t.Run("Filesystem Adapter", testFailToWriteOutOfOrder(aFileSystemAdapter))
}

func TestBlockPersistenceContract_ReturnsTwoBlocks(t *testing.T) {
	t.Run("In Memory Adapter", testWritesAndReturnsTwoBlocks(anInMemoryAdapter))
	t.Run("Filesystem Adapter", testWritesAndReturnsTwoBlocks(aFileSystemAdapter))
}

func testWritesAndReturnsTwoBlocks(factory func() (adapter.BlockPersistence, func())) func(t *testing.T) {
	return func(t *testing.T) {
		adapter, cleanup := factory()
		defer cleanup()

		block1 := builders.BlockPair().WithHeight(1).Build()
		block2 := builders.BlockPair().WithHeight(2).WithPrevBlock(block1).Build()

		err := adapter.WriteNextBlock(block1)
		require.NoError(t, err)
		err = adapter.WriteNextBlock(block2)
		require.NoError(t, err)

		require.NoError(t, adapter.ScanBlocks(1, 2, func(first primitives.BlockHeight, page []*protocol.BlockPairContainer) bool {
			test.RequireCmpEqual(t, block1, page[0], "block 1 did not match")
			test.RequireCmpEqual(t, block2, page[1], "block 2 did not match")

			return false
		}))
	}
}

func testWritesBlockAndRetrieves(factory func() (adapter.BlockPersistence, func())) func(t *testing.T) {
	return func(t *testing.T) {
		adapter, cleanup := factory()
		defer cleanup()

		block := builders.BlockPair().WithHeight(1).Build()
		err := adapter.WriteNextBlock(block)
		require.NoError(t, err)

		lastBlock, err := adapter.GetLastBlock()
		require.NoError(t, err)
		test.RequireCmpEqual(t, block, lastBlock, "expected to retrieve same block written")
	}
}

func testFailToWriteOutOfOrder(factory func() (adapter.BlockPersistence, func())) func(t *testing.T) {
	return func(t *testing.T) {
		adapter, cleanup := factory()
		defer cleanup()

		err := adapter.WriteNextBlock(builders.BlockPair().WithHeight(2).Build())
		require.Error(t, err)
	}
}

func anInMemoryAdapter() (adapter.BlockPersistence, func()) {
	return NewInMemoryBlockPersistence(log.GetLogger(), metric.NewRegistry()), func() {}
}

func aFileSystemAdapter() (adapter.BlockPersistence, func()) {
	dirName, err := ioutil.TempDir("", "contract_test_block_persist")
	if err != nil {
		panic(err)
	}
	cleanup := func() {
		os.RemoveAll(dirName)
	}
	return NewFilesystemBlockPersistence(dirName), cleanup
}

func NewFilesystemBlockPersistence(dataDir string) adapter.BlockPersistence {
	return &FilesystemBlockPersistence{
		dataFile: struct {
			sync.RWMutex
			dataDir   string
			topHeight primitives.BlockHeight
		}{
			dataDir:   dataDir,
			topHeight: 0,
		},
	}
}

type FilesystemBlockPersistence struct {
	dataFile struct {
		sync.RWMutex
		dataDir   string
		topHeight primitives.BlockHeight
	}
}

func (f *FilesystemBlockPersistence) WriteNextBlock(blockPair *protocol.BlockPairContainer) error {
	f.dataFile.Lock()
	defer f.dataFile.Unlock()

	bh := blockPair.ResultsBlock.Header.BlockHeight()
	if bh != f.dataFile.topHeight+1 {
		return fmt.Errorf("attempt to write block %d out of order. current top height is %d", bh, f.dataFile.topHeight)
	}

	file, err := os.Create(f.dataFile.dataDir + "/blocks")
	if err != nil {
		return errors.Wrap(err, "failed to open blocks file for writing")
	}
	coded, err := codec.EncodeBlockPair(blockPair)
	if err != nil {
		return errors.Wrap(err, "failed to serialize block")
	}

	for _, arr := range coded {
		chunkSize := make([]byte, 4)
		binary.LittleEndian.PutUint32(chunkSize, uint32(len(arr)))
		n, err := file.Write(chunkSize)
		if err != nil {
			return errors.Wrap(err, "failed to write bytes to blocks file")
		}
		if n != len(chunkSize) {
			return fmt.Errorf("wrote less bytes than requested to blocks file")
		}
		n, err = file.Write(arr)
		if err != nil {
			return errors.Wrap(err, "failed to write bytes to blocks file")
		}
		if n != len(arr) {
			return fmt.Errorf("wrote less bytes than requested to blocks file")
		}
	}
	err = file.Sync()
	if err != nil {
		return errors.Wrap(err, "failed to flush blocks file to disk")
	}
	f.dataFile.topHeight++
	return nil
}

func (*FilesystemBlockPersistence) ScanBlocks(from primitives.BlockHeight, pageSize uint8, f adapter.CursorFunc) error {
	panic("implement me")
}

func (*FilesystemBlockPersistence) GetLastBlockHeight() (primitives.BlockHeight, error) {
	panic("implement me")
}

func (f *FilesystemBlockPersistence) GetLastBlock() (*protocol.BlockPairContainer, error) {
	f.dataFile.RLock()
	defer f.dataFile.RUnlock()

	var chunks [][]byte
	file, err := os.Open(f.dataFile.dataDir + "/blocks")
	if err != nil {
		return nil, errors.Wrap(err, "failed to open blocks file for reading")
	}

	for {
		chunkSize := make([]byte, 4)
		n, err := file.Read(chunkSize)
		if n == 0 { // EOF
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "failed to read bytes from blocks file")
		}
		if n != len(chunkSize) {
			return nil, fmt.Errorf("read less bytes than requested from blocks file")
		}
		bytesToRead := binary.LittleEndian.Uint32(chunkSize)
		chunk := make([]byte, bytesToRead)
		n, err = file.Read(chunk)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read bytes from blocks file")
		}
		if n != len(chunk) {
			return nil, fmt.Errorf("read less bytes than requested from blocks file")
		}
		chunks = append(chunks, chunk)
	}
	result, err := codec.DecodeBlockPair(chunks)
	if err != nil {
		return nil, errors.Wrap(err, "failed to deserialize block")
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

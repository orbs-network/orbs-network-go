package adapter

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
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
	return adapter.NewFilesystemBlockPersistence(dirName), cleanup
}

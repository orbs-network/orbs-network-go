package adapter

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
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
	"time"
)

func TestBlockPersistenceContract_WritesBlockAndRetrieves(t *testing.T) {
	ctrlRand := test.NewControlledRand(t)
	for i := 1; i <= 10; i++ {
		blocks := buildRandomBlockChain(ctrlRand.Int31n(100)+1, ctrlRand)
		t.Run(fmt.Sprintf("In Memory Adapter %d", i), testWritesBlockAndRetrieves(anInMemoryAdapter, blocks))
		t.Run(fmt.Sprintf("Filesystem Adapter %d", i), testWritesBlockAndRetrieves(aFileSystemAdapter, blocks))
	}
}

func TestBlockPersistenceContract_FailToWriteOutOfOrder(t *testing.T) {
	t.Run("In Memory Adapter", testFailToWriteOutOfOrder(anInMemoryAdapter))
	t.Run("Filesystem Adapter", testFailToWriteOutOfOrder(aFileSystemAdapter))
}

func TestBlockPersistenceContract_ReturnsTwoBlocks(t *testing.T) {
	t.Run("In Memory Adapter", testWritesAndReturnsTwoBlocks(anInMemoryAdapter))
	t.Run("Filesystem Adapter", testWritesAndReturnsTwoBlocks(aFileSystemAdapter))
}

func TestBlockPersistenceContract_BlockTrackerBlocksUntilRequestedHeight(t *testing.T) {
	t.Run("In Memory Adapter", testBlockTrackerBlocksUntilRequestedHeight(anInMemoryAdapter))
	t.Run("Filesystem Adapter", testBlockTrackerBlocksUntilRequestedHeight(aFileSystemAdapter))
}

func TestBlockPersistenceContract_ReturnsLastBlockHeight(t *testing.T) {
	t.Run("In Memory Adapter", testReturnsLastBlockHeight(anInMemoryAdapter))
	t.Run("Filesystem Adapter", testReturnsLastBlockHeight(aFileSystemAdapter))
}

func TestBlockPersistenceContract_ReturnsTransactionsAndResultsBlock(t *testing.T) {
	t.Run("In Memory Adapter", testReturnsTransactionsAndResultsBlock(anInMemoryAdapter))
	t.Run("Filesystem Adapter", testReturnsTransactionsAndResultsBlock(aFileSystemAdapter))
}

func TestBlockPersistenceContract_ReturnsBlockByTx(t *testing.T) {
	t.Run("In Memory Adapter", testReturnsBlockByTx(anInMemoryAdapter))
	t.Run("Filesystem Adapter", testReturnsBlockByTx(aFileSystemAdapter))
}

func testReturnsBlockByTx(factory func() (adapter.BlockPersistence, func())) func(t *testing.T) {
	return func(t *testing.T) {
		adapter, cleanup := factory()
		defer cleanup()

		blocks := []*protocol.BlockPairContainer{
			builders.BlockPair().WithHeight(1).WithTransactions(1).WithReceiptsForTransactions().Build(),
			builders.BlockPair().WithHeight(2).WithTransactions(7).WithReceiptsForTransactions().Build(),
			builders.BlockPair().WithHeight(3).WithTransactions(1).WithReceiptsForTransactions().Build(),
		}

		for _, b := range blocks {
			err := adapter.WriteNextBlock(b)
			require.NoError(t, err)
		}

		tx := blocks[1].TransactionsBlock.SignedTransactions[6].Transaction()

		readBlock, txIndex, err := adapter.GetBlockByTx(digest.CalcTxHash(tx), blocks[0].ResultsBlock.Header.Timestamp(), blocks[2].ResultsBlock.Header.Timestamp())
		require.NoError(t, err)
		require.EqualValues(t, 6, txIndex)
		test.RequireCmpEqual(t, readBlock, blocks[1])
	}
}

func testReturnsTransactionsAndResultsBlock(factory func() (adapter.BlockPersistence, func())) func(t *testing.T) {
	return func(t *testing.T) {
		adapter, cleanup := factory()
		defer cleanup()

		blocks := []*protocol.BlockPairContainer{
			builders.BlockPair().WithHeight(1).WithTransactions(1).WithReceiptsForTransactions().Build(),
			builders.BlockPair().WithHeight(2).WithTransactions(2).WithReceiptsForTransactions().Build(),
			builders.BlockPair().WithHeight(3).WithTransactions(3).WithReceiptsForTransactions().Build(),
		}

		for _, b := range blocks {
			err := adapter.WriteNextBlock(b)
			require.NoError(t, err)
		}

		readTxBlock, err := adapter.GetTransactionsBlock(2)
		require.NoError(t, err)

		readResultsBlock, err := adapter.GetResultsBlock(2)
		require.NoError(t, err)

		require.Len(t, readTxBlock.SignedTransactions, 2)
		test.RequireCmpEqual(t, readTxBlock, blocks[1].TransactionsBlock)

		require.Len(t, readResultsBlock.TransactionReceipts, 2)
		test.RequireCmpEqual(t, readResultsBlock, blocks[1].ResultsBlock)

	}
}

func testReturnsLastBlockHeight(factory func() (adapter.BlockPersistence, func())) func(t *testing.T) {
	return func(t *testing.T) {
		adapter, cleanup := factory()
		defer cleanup()

		h, err := adapter.GetLastBlockHeight()
		require.NoError(t, err)
		require.EqualValues(t, 0, h)

		// write block height 1
		err = adapter.WriteNextBlock(builders.BlockPair().WithHeight(1).Build())
		require.NoError(t, err)

		h, err = adapter.GetLastBlockHeight()
		require.NoError(t, err)
		require.EqualValues(t, 1, h)
	}
}

func testBlockTrackerBlocksUntilRequestedHeight(factory func() (adapter.BlockPersistence, func())) func(t *testing.T) {
	return func(t *testing.T) {
		adapter, cleanup := factory()
		defer cleanup()

		// block until timeout before block is written
		shortDeadlineCtx, _ := context.WithTimeout(context.Background(), 5*time.Millisecond)
		err := adapter.GetBlockTracker().WaitForBlock(shortDeadlineCtx, 1)
		require.Error(t, err)

		// write block height 1
		err = adapter.WriteNextBlock(builders.BlockPair().WithHeight(1).Build())
		require.NoError(t, err)

		// block tracker returns from wait immediately without error
		shortDeadlineCtx, _ = context.WithTimeout(context.Background(), 5*time.Millisecond)
		err = adapter.GetBlockTracker().WaitForBlock(shortDeadlineCtx, 1)
		require.NoError(t, err)
	}
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

func testWritesBlockAndRetrieves(factory func() (adapter.BlockPersistence, func()), blocks []*protocol.BlockPairContainer) func(t *testing.T) {
	return func(t *testing.T) {
		adapter, cleanup := factory()
		defer cleanup()

		for _, b := range blocks {
			err := adapter.WriteNextBlock(b)
			require.NoError(t, err)
		}

		// test ScanBlocks
		readBlocks := make([]*protocol.BlockPairContainer, 0, len(blocks))
		err := adapter.ScanBlocks(1, 9, func(first primitives.BlockHeight, page []*protocol.BlockPairContainer) (wantsMore bool) {
			readBlocks = append(readBlocks, page...)
			return true
		})
		require.NoError(t, err)
		require.Equal(t, len(readBlocks), len(blocks))
		test.RequireCmpEqual(t, blocks, readBlocks)

		// test GetLastBlock
		lastBlock, err := adapter.GetLastBlock()
		require.NoError(t, err)
		test.RequireCmpEqual(t, blocks[len(blocks)-1], lastBlock, "expected to retrieve an identical block from storage")
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

func buildRandomBlockChain(numBlocks int32, ctrlRand *test.ControlledRand) []*protocol.BlockPairContainer {
	blocks := make([]*protocol.BlockPairContainer, 0, numBlocks)

	var prev *protocol.BlockPairContainer
	for bi := 1; bi <= cap(blocks); bi++ {
		newBlock := newRandomBlockBuilder(primitives.BlockHeight(bi), ctrlRand, prev)
		blocks = append(blocks, newBlock)
		prev = newBlock
	}
	return blocks
}

func newRandomBlockBuilder(h primitives.BlockHeight, ctrlRand *test.ControlledRand, prev *protocol.BlockPairContainer) *protocol.BlockPairContainer {
	builder := builders.BlockPair().
		WithHeight(h).
		WithTransactions(ctrlRand.Uint32() % 100).
		WithReceiptsForTransactions().
		WithLeanHelixBlockProof()
	if prev != nil {
		builder.WithPrevBlock(prev)
	}
	return builder.Build()
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
	persistence, err := adapter.NewFilesystemBlockPersistence(dirName, log.GetLogger(), metric.NewRegistry())
	if err != nil {
		panic(err)
	}
	return persistence, cleanup
}

// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/crypto-lib-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter/filesystem"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter/memory"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func withEachAdapter(t *testing.T, testFunc func(t *testing.T, adapter adapter.BlockPersistence)) {
	if testing.Short() {
		t.Skip("Skipping contract tests in short mode")
	}

	t.Run("File System Persistence", func(t *testing.T) {
		with.Logging(t, func(harness *with.LoggingHarness) {
			adapter, cleanup := newFilesystemAdapter(harness.Logger)
			defer cleanup()
			testFunc(t, adapter)
		})
	})

	t.Run("In-Memory Persistence", func(t *testing.T) {
		with.Logging(t, func(harness *with.LoggingHarness) {
			testFunc(t, newInMemoryAdapter(harness.Logger))
		})
	})
}

func TestBlockPersistenceContract_GetLastBlockWhenNoneExistReturnsNilNoError(t *testing.T) {
	withEachAdapter(t, func(t *testing.T, adapter adapter.BlockPersistence) {
		lastBlock, err := adapter.GetLastBlock()
		require.NoError(t, err)
		require.Nil(t, lastBlock)
	})
}

func TestBlockPersistenceContract_WritesBlockAndRetrieves(t *testing.T) {
	ctrlRand := rand.NewControlledRand(t)
	for i := 1; i <= 10; i++ {

		blocks := builders.RandomizedBlockChain(ctrlRand.Int31n(100)+10, ctrlRand)

		withEachAdapter(t, func(t *testing.T, adapter adapter.BlockPersistence) {
			for _, b := range blocks {
				added, pHeight, err := adapter.WriteNextBlock(b)
				require.NoError(t, err, "write should succeed")
				require.True(t, added, "block should actually be added (it's not duplicate)")
				require.EqualValues(t, b.TransactionsBlock.Header.BlockHeight(), pHeight, "expected height to be reported correctly")
			}

			// test ScanBlocks
			skip := 4
			readBlocks := make([]*protocol.BlockPairContainer, 0, len(blocks))
			err := adapter.ScanBlocks(primitives.BlockHeight(skip)+1, 7, func(first primitives.BlockHeight, page []*protocol.BlockPairContainer) (wantsMore bool) {
				readBlocks = append(readBlocks, page...)
				return true
			})
			require.NoError(t, err)
			require.Equal(t, len(blocks)-skip, len(readBlocks))
			test.RequireCmpEqual(t, blocks[skip:], readBlocks)

			// test GetLastBlock
			lastBlock, err := adapter.GetLastBlock()
			require.NoError(t, err)
			test.RequireCmpEqual(t, blocks[len(blocks)-1], lastBlock, "expected to retrieve an identical block from storage")
		})
	}
}

func TestBlockPersistenceContract_ReadUnknownBlocksReturnsError(t *testing.T) {
	withEachAdapter(t, func(t *testing.T, adapter adapter.BlockPersistence) {
		err := adapter.ScanBlocks(1, 1, func(first primitives.BlockHeight, page []*protocol.BlockPairContainer) (wantsMore bool) {
			t.Fatal("expected cursorFunc never to be invoked if requested block height is not in storage")
			return false
		})
		require.Error(t, err)
	})
}

func TestBlockPersistenceContract_WriteOutOfOrderFuture_Succeeds(t *testing.T) {
	withEachAdapter(t, func(t *testing.T, adapter adapter.BlockPersistence) {
		added, persistedHeight, err := adapter.WriteNextBlock(builders.BlockPair().WithHeight(1).Build())
		require.NoError(t, err)
		require.True(t, added)
		require.EqualValues(t, 1, persistedHeight)

		added, persistedHeight, err = adapter.WriteNextBlock(builders.BlockPair().WithHeight(3).Build())
		require.NoError(t, err)
		require.True(t, added, "persistence storage should support out of order writes")
		require.EqualValues(t, 3, persistedHeight, "persisted height should be reported correctly as lastSynced block height")
	})
}

func TestBlockPersistenceContract_WriteOutOfOrderPast_NotFailsWithoutError(t *testing.T) {
	withEachAdapter(t, func(t *testing.T, adapter adapter.BlockPersistence) {
		now := time.Now()
		block1 := builders.BlockPair().WithHeight(1).WithTransactions(1).WithBlockCreated(now).Build()
		block2 := builders.BlockPair().WithHeight(2).WithPrevBlock(block1).Build()
		otherBlockDifferentTimestamp := builders.BlockPair().WithHeight(1).WithBlockCreated(now.Add(1 * time.Second)).Build()
		otherBlockDifferentTransactionsBlock := builders.BlockPair().WithHeight(1).WithTransactions(2).WithBlockCreated(now).Build()
		otherBlockDifferentResultsBlock := builders.BlockPair().WithHeight(1).WithTransactions(1).WithReceiptsForTransactions().WithBlockCreated(now).Build()

		added, pHeight, err := adapter.WriteNextBlock(block1)
		require.NoError(t, err, "write should succeed")
		require.True(t, added, "block should actually be added (it's not duplicate)")
		require.EqualValues(t, 1, pHeight)

		added, pHeight, err = adapter.WriteNextBlock(block2)
		require.NoError(t, err, "write should succeed")
		require.True(t, added, "block should actually be added (it's not duplicate)")
		require.EqualValues(t, 2, pHeight)

		// write past blocks:
		added, pHeight, err = adapter.WriteNextBlock(block1)
		require.NoError(t, err, "expected past blocks to be ignored")
		require.False(t, added, "block should not be added (due to failure)")
		require.EqualValues(t, 2, pHeight, "expected persisted height to not change")

		added, pHeight, err = adapter.WriteNextBlock(otherBlockDifferentTimestamp)
		require.NoError(t, err, "expected past blocks to be ignored without checking content")
		require.False(t, added, "block should not be added (due to failure)")
		require.EqualValues(t, 2, pHeight, "expected persisted height to not change")

		added, pHeight, err = adapter.WriteNextBlock(otherBlockDifferentResultsBlock)
		require.NoError(t, err, "expected past blocks to be ignored without checking content")
		require.False(t, added, "block should not be added (due to failure)")
		require.EqualValues(t, 2, pHeight, "expected persisted height to not change")

		added, pHeight, err = adapter.WriteNextBlock(otherBlockDifferentTransactionsBlock)
		require.NoError(t, err, "expected past blocks to be ignored without checking content")
		require.False(t, added, "block should not be added (due to failure)")
		require.EqualValues(t, 2, pHeight, "expected persisted height to not change")
	})
}

func TestBlockPersistenceContract_ReturnsTwoBlocks(t *testing.T) {
	withEachAdapter(t, func(t *testing.T, adapter adapter.BlockPersistence) {
		block1 := builders.BlockPair().WithHeight(1).Build()
		block2 := builders.BlockPair().WithHeight(2).WithPrevBlock(block1).Build()

		added, _, err := adapter.WriteNextBlock(block1)
		require.NoError(t, err, "write should succeed")
		require.True(t, added, "block should actually be added (it's not duplicate)")
		added, _, err = adapter.WriteNextBlock(block2)
		require.NoError(t, err, "write should succeed")
		require.True(t, added, "block should actually be added (it's not duplicate)")

		require.NoError(t, adapter.ScanBlocks(1, 2, func(first primitives.BlockHeight, page []*protocol.BlockPairContainer) bool {
			test.RequireCmpEqual(t, block1, page[0], "block 1 did not match")
			test.RequireCmpEqual(t, block2, page[1], "block 2 did not match")

			return false
		}))
	})
}

func TestBlockPersistenceContract_BlockTrackerBlocksUntilRequestedHeight(t *testing.T) {
	withEachAdapter(t, func(t *testing.T, adapter adapter.BlockPersistence) {

		// block until timeout before block is written
		shortDeadlineCtx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
		defer cancel()
		err := adapter.GetBlockTracker().WaitForBlock(shortDeadlineCtx, 1)
		require.Error(t, err, "expected timeout to expire when requested block height not available")

		// write block height 1
		added, newHeight, err := adapter.WriteNextBlock(builders.BlockPair().WithHeight(1).Build())
		require.NoError(t, err, "write should succeed")
		require.True(t, added, "block should actually be added (it's not duplicate)")
		require.EqualValues(t, 1, newHeight, "expected persisted height to be 1")

		// block tracker returns from wait immediately without error
		shortDeadlineCtx, cancel = context.WithTimeout(context.Background(), 5*time.Millisecond)
		defer cancel()
		err = adapter.GetBlockTracker().WaitForBlock(shortDeadlineCtx, 1)
		require.NoError(t, err)
	})
}

func TestBlockPersistenceContract_ReturnsLastBlockHeight(t *testing.T) {
	withEachAdapter(t, func(t *testing.T, adapter adapter.BlockPersistence) {
		h, err := adapter.GetLastBlockHeight()
		require.NoError(t, err)
		require.EqualValues(t, 0, h)

		// write block height 1
		added, persistedHeight, err := adapter.WriteNextBlock(builders.BlockPair().WithHeight(1).Build())
		require.NoError(t, err, "write should succeed")
		require.True(t, added, "block should actually be added (it's not duplicate)")
		require.EqualValues(t, 1, persistedHeight, "expected persisted height to be 1")

		h, err = adapter.GetLastBlockHeight()
		require.NoError(t, err)
		require.EqualValues(t, 1, h)
	})
}

func TestBlockPersistenceContract_ReturnsTransactionsAndResultsBlock(t *testing.T) {
	withEachAdapter(t, func(t *testing.T, adapter adapter.BlockPersistence) {
		blocks := []*protocol.BlockPairContainer{
			builders.BlockPair().WithHeight(1).WithTransactions(1).WithReceiptsForTransactions().Build(),
			builders.BlockPair().WithHeight(2).WithTransactions(2).WithReceiptsForTransactions().Build(),
			builders.BlockPair().WithHeight(3).WithTransactions(3).WithReceiptsForTransactions().Build(),
		}

		for _, b := range blocks {
			added, _, err := adapter.WriteNextBlock(b)
			require.NoError(t, err)
			require.True(t, added)
		}

		readTxBlock, err := adapter.GetTransactionsBlock(2)
		require.NoError(t, err)

		readResultsBlock, err := adapter.GetResultsBlock(2)
		require.NoError(t, err)

		require.Len(t, readTxBlock.SignedTransactions, 2)
		test.RequireCmpEqual(t, readTxBlock, blocks[1].TransactionsBlock)

		require.Len(t, readResultsBlock.TransactionReceipts, 2)
		test.RequireCmpEqual(t, readResultsBlock, blocks[1].ResultsBlock)
	})
}

func TestBlockPersistenceContract_ReturnsBlockByTx(t *testing.T) {
	withEachAdapter(t, func(t *testing.T, adapter adapter.BlockPersistence) {
		blocks := []*protocol.BlockPairContainer{
			builders.BlockPair().WithHeight(1).WithTransactions(1).WithReceiptsForTransactions().Build(),
			builders.BlockPair().WithHeight(2).WithTransactions(7).WithReceiptsForTransactions().Build(),
			builders.BlockPair().WithHeight(3).WithTransactions(1).WithReceiptsForTransactions().Build(),
		}

		for _, b := range blocks {
			added, _, err := adapter.WriteNextBlock(b)
			require.NoError(t, err, "write should succeed")
			require.True(t, added, "block should actually be added (it's not duplicate)")
		}

		tx := blocks[1].TransactionsBlock.SignedTransactions[6].Transaction()

		readBlock, txIndex, err := adapter.GetBlockByTx(digest.CalcTxHash(tx), blocks[0].ResultsBlock.Header.Timestamp(), blocks[2].ResultsBlock.Header.Timestamp())
		require.NoError(t, err)
		require.EqualValues(t, 6, txIndex)
		test.RequireCmpEqual(t, readBlock, blocks[1])
	})
}

func TestReturnTransactionReceipt(t *testing.T) {
	withEachAdapter(t, func(t *testing.T, adapter adapter.BlockPersistence) {

		blocks := []*protocol.BlockPairContainer{
			builders.BlockPair().WithHeight(1).WithTransactions(13).WithReceiptsForTransactions().WithTimestampAheadBy(2 * time.Second).Build(),
			builders.BlockPair().WithHeight(2).WithTransactions(07).WithReceiptsForTransactions().WithTimestampAheadBy(4 * time.Second).Build(),
			builders.BlockPair().WithHeight(3).WithTransactions(01).WithReceiptsForTransactions().WithTimestampAheadBy(6 * time.Second).Build(),
		}

		for _, b := range blocks {
			added, _, err := adapter.WriteNextBlock(b)
			require.NoError(t, err, "write should succeed")
			require.True(t, added, "block should actually be added (it's not duplicate)")
		}

		second := primitives.TimestampNano(1 * time.Second)
		block := blocks[1]
		blockTimestamp := block.ResultsBlock.Header.Timestamp()
		txIndex := 6
		tx := block.TransactionsBlock.SignedTransactions[txIndex].Transaction()

		retrievedBlock, retrievedTxIndex, err := adapter.GetBlockByTx(digest.CalcTxHash(tx), blockTimestamp-second, blockTimestamp+second)
		require.NoError(t, err)
		test.RequireCmpEqual(t, block, retrievedBlock, "expected correct block to be retrieved")
		require.EqualValues(t, txIndex, retrievedTxIndex, "expected correct tx index to be retrieved")
	})
}

func newInMemoryAdapter(logger log.Logger) adapter.BlockPersistence {
	return memory.NewBlockPersistence(logger, metric.NewRegistry())
}

func newFilesystemAdapter(logger log.Logger) (adapter.BlockPersistence, func()) {
	conf := newTempFileConfig()

	persistence, err := filesystem.NewBlockPersistence(conf, logger, metric.NewRegistry())
	if err != nil {
		panic(err.Error())
	}

	cleanup := func() {
		persistence.GracefulShutdown(context.Background())
		_ = os.RemoveAll(conf.BlockStorageFileSystemDataDir()) // ignore errors - nothing to do
	}

	return persistence, cleanup
}

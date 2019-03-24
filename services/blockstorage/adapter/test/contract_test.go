// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter/filesystem"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter/memory"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func withEachAdapter(t *testing.T, testFunc func(t *testing.T, adapter adapter.BlockPersistence)) {
	if testing.Short() {
		t.Skip("Skipping contract tests in short mode")
	}
	adapters := []*adapterUnderTest{
		newInMemoryAdapter(t),
		newFilesystemAdapter(t),
	}
	for _, a := range adapters {
		t.Run(a.name, func(t *testing.T) {
			defer a.cleanup()
			testFunc(t, a.adapter)
		})
	}
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
				added, err := adapter.WriteNextBlock(b)
				require.NoError(t, err, "write should succeed")
				require.True(t, added, "block should actually be added (it's not duplicate)")
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

func TestBlockPersistenceContract_WriteOutOfOrderFuture_Fails(t *testing.T) {
	withEachAdapter(t, func(t *testing.T, adapter adapter.BlockPersistence) {
		added, err := adapter.WriteNextBlock(builders.BlockPair().WithHeight(2).Build())
		require.Error(t, err, "write should fail")
		require.False(t, added, "block should not be added (due to failure)")
	})
}

func TestBlockPersistenceContract_WriteOutOfOrderPast_NotFailsWhenBlockIdentical(t *testing.T) {
	withEachAdapter(t, func(t *testing.T, adapter adapter.BlockPersistence) {
		block1 := builders.BlockPair().WithHeight(1).Build()
		block2 := builders.BlockPair().WithHeight(2).WithPrevBlock(block1).Build()

		added, err := adapter.WriteNextBlock(block1)
		require.NoError(t, err, "write should succeed")
		require.True(t, added, "block should actually be added (it's not duplicate)")
		added, err = adapter.WriteNextBlock(block2)
		require.NoError(t, err, "write should succeed")
		require.True(t, added, "block should actually be added (it's not duplicate)")

		added, err = adapter.WriteNextBlock(block1)
		require.NoError(t, err, "write should succeed")
		require.False(t, added, "block should not be added though (it's duplicate)")
	})
}

func TestBlockPersistenceContract_WriteOutOfOrderPast_FailsWhenBlockDifferent(t *testing.T) {
	t.Skip("fails with In_Memory_Adapter and should be fixed") // TODO(v1): fix
	withEachAdapter(t, func(t *testing.T, adapter adapter.BlockPersistence) {
		now := time.Now()
		block1 := builders.BlockPair().WithHeight(1).WithBlockCreated(now).Build()
		block2 := builders.BlockPair().WithHeight(2).WithPrevBlock(block1).Build()
		otherBlock1 := builders.BlockPair().WithHeight(1).WithBlockCreated(now.Add(1 * time.Second)).Build()

		added, err := adapter.WriteNextBlock(block1)
		require.NoError(t, err, "write should succeed")
		require.True(t, added, "block should actually be added (it's not duplicate)")
		added, err = adapter.WriteNextBlock(block2)
		require.NoError(t, err, "write should succeed")
		require.True(t, added, "block should actually be added (it's not duplicate)")

		added, err = adapter.WriteNextBlock(otherBlock1)
		require.Error(t, err, "write of a different old block should fail")
		require.False(t, added, "block should not be added (due to failure)")
	})
}

func TestBlockPersistenceContract_ReturnsTwoBlocks(t *testing.T) {
	withEachAdapter(t, func(t *testing.T, adapter adapter.BlockPersistence) {
		block1 := builders.BlockPair().WithHeight(1).Build()
		block2 := builders.BlockPair().WithHeight(2).WithPrevBlock(block1).Build()

		added, err := adapter.WriteNextBlock(block1)
		require.NoError(t, err, "write should succeed")
		require.True(t, added, "block should actually be added (it's not duplicate)")
		added, err = adapter.WriteNextBlock(block2)
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
		shortDeadlineCtx, _ := context.WithTimeout(context.Background(), 5*time.Millisecond)
		err := adapter.GetBlockTracker().WaitForBlock(shortDeadlineCtx, 1)
		require.Error(t, err)

		// write block height 1
		added, err := adapter.WriteNextBlock(builders.BlockPair().WithHeight(1).Build())
		require.NoError(t, err, "write should succeed")
		require.True(t, added, "block should actually be added (it's not duplicate)")

		// block tracker returns from wait immediately without error
		shortDeadlineCtx, _ = context.WithTimeout(context.Background(), 5*time.Millisecond)
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
		added, err := adapter.WriteNextBlock(builders.BlockPair().WithHeight(1).Build())
		require.NoError(t, err, "write should succeed")
		require.True(t, added, "block should actually be added (it's not duplicate)")

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
			added, err := adapter.WriteNextBlock(b)
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
			added, err := adapter.WriteNextBlock(b)
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

type adapterUnderTest struct {
	name    string
	adapter adapter.BlockPersistence
	cleanup func()
}

func newInMemoryAdapter(tb testing.TB) *adapterUnderTest {
	return &adapterUnderTest{
		name:    "In Memory Adapter",
		adapter: memory.NewBlockPersistence(log.DefaultTestingLogger(tb), metric.NewRegistry()),
		cleanup: func() {},
	}
}

func newFilesystemAdapter(tb testing.TB) *adapterUnderTest {
	ctx, cancel := context.WithCancel(context.Background())

	conf := newTempFileConfig()
	cleanup := func() {
		cancel()
		_ = os.RemoveAll(conf.BlockStorageFileSystemDataDir()) // ignore errors - nothing to do
	}

	logger := log.DefaultTestingLogger(tb)
	persistence, err := filesystem.NewBlockPersistence(ctx, conf, logger, metric.NewRegistry())
	if err != nil {
		panic(err.Error())
	}

	return &adapterUnderTest{
		name:    "File System Adapter",
		adapter: persistence,
		cleanup: cleanup,
	}
}

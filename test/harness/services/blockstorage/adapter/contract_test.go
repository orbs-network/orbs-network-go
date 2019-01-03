package adapter

import (
	"context"
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

func withEachAdapter(t *testing.T, testFunc func(t *testing.T, adapter adapter.BlockPersistence)) {
	adapters := []*adapterUnderTest{
		newInMemoryAdapter(),
		newFilesystemAdapter(),
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
	ctrlRand := test.NewControlledRand(t)
	for i := 1; i <= 10; i++ {

		blocks := builders.RandomizedBlockChain(ctrlRand.Int31n(100)+1, ctrlRand)

		withEachAdapter(t, func(t *testing.T, adapter adapter.BlockPersistence) {
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
		})
	}
}

func TestBlockPersistenceContract_FailToWriteOutOfOrder(t *testing.T) {
	withEachAdapter(t, func(t *testing.T, adapter adapter.BlockPersistence) {
		err := adapter.WriteNextBlock(builders.BlockPair().WithHeight(2).Build())
		require.Error(t, err)
	})
}

func TestBlockPersistenceContract_ReturnsTwoBlocks(t *testing.T) {
	withEachAdapter(t, func(t *testing.T, adapter adapter.BlockPersistence) {
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
	})
}

func TestBlockPersistenceContract_BlockTrackerBlocksUntilRequestedHeight(t *testing.T) {
	withEachAdapter(t, func(t *testing.T, adapter adapter.BlockPersistence) {

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
	})
}

func TestBlockPersistenceContract_ReturnsLastBlockHeight(t *testing.T) {
	withEachAdapter(t, func(t *testing.T, adapter adapter.BlockPersistence) {
		h, err := adapter.GetLastBlockHeight()
		require.NoError(t, err)
		require.EqualValues(t, 0, h)

		// write block height 1
		err = adapter.WriteNextBlock(builders.BlockPair().WithHeight(1).Build())
		require.NoError(t, err)

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
			err := adapter.WriteNextBlock(b)
			require.NoError(t, err)
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

func newInMemoryAdapter() *adapterUnderTest {
	return &adapterUnderTest{
		name:    "In Memory Adapter",
		adapter: NewInMemoryBlockPersistence(log.GetLogger(), metric.NewRegistry()),
		cleanup: func() {},
	}
}

func newFilesystemAdapter() *adapterUnderTest {
	ctx, cancel := context.WithCancel(context.Background())

	conf := newLocalConfig()
	cleanup := func() {
		cancel()
		_ = os.RemoveAll(conf.BlockStorageDataDir()) // ignore errors - nothing to do
	}

	persistence, err := adapter.NewFilesystemBlockPersistence(ctx, conf, log.GetLogger(), metric.NewRegistry())
	if err != nil {
		panic(err)
	}

	return &adapterUnderTest{
		name:    "File System Adapter",
		adapter: persistence,
		cleanup: cleanup,
	}
}

type localConfig struct {
	dir string
}

func newLocalConfig() *localConfig {
	dirName, err := ioutil.TempDir("", "contract_test_block_persist")
	if err != nil {
		panic(err)
	}
	return &localConfig{
		dir: dirName,
	}
}
func (l *localConfig) BlockStorageDataDir() string {
	return l.dir
}

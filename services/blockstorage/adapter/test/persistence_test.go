// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter/memory"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPersistenceAdapter_CanAccessBlocksOutOfOrder(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Integration tests in short mode")
	}
	with.Logging(t, func(harness *with.LoggingHarness) {
		ctrlRand := rand.NewControlledRand(t)
		blocks := builders.RandomizedBlockChain(50, ctrlRand)

		conf := newTempFileConfig()
		defer conf.cleanDir()

		adapter1, close1, err := NewFilesystemAdapterDriver(harness.Logger, conf)
		require.NoError(t, err)

		for _, block := range blocks { // write some blocks
			added, pHeight, err := adapter1.WriteNextBlock(block)
			require.NoError(t, err)
			require.True(t, added)
			require.EqualValues(t, block.TransactionsBlock.Header.BlockHeight(), pHeight)
		}

		requireCanReadAllBlocksInRandomOrder(t, adapter1, blocks, ctrlRand)
		close1()

		adapter2, close2, err := NewFilesystemAdapterDriver(harness.Logger, conf)
		require.NoError(t, err)

		requireCanReadAllBlocksInRandomOrder(t, adapter2, blocks, ctrlRand)
		close2()
	})
}

func requireCanReadAllBlocksInRandomOrder(t *testing.T, adapter adapter.BlockPersistence, blocks []*protocol.BlockPairContainer, ctrlRand *rand.ControlledRand) {
	for _, i := range ctrlRand.Perm(len(blocks)) { // read each block out of order
		h := primitives.BlockHeight(i + 1)
		block, err := readOneBlock(adapter, h)
		test.RequireCmpEqual(t, blocks[i], block, "expected to succeed in reading block at height %v", h)
		t.Logf("successfully read block height %v", i+1)
		require.NoError(t, err)
	}
}

func readOneBlock(fsa adapter.BlockPersistence, h primitives.BlockHeight) (*protocol.BlockPairContainer, error) {
	var block *protocol.BlockPairContainer
	err := fsa.ScanBlocks(h, 1, func(first primitives.BlockHeight, page []*protocol.BlockPairContainer) (wantsMore bool) {
		block = page[0]
		return false
	})
	return block, err
}

func testBlockPersistenceWriteLogicWithAdapter(t *testing.T, persistence adapter.BlockPersistence, blocks []*protocol.BlockPairContainer) {
	var wroteBlock bool
	var err error
	var lastBlockHeight primitives.BlockHeight
	var block *protocol.BlockPairContainer

	for i := 0; i < 10; i++ {
		block = blocks[i]
		wroteBlock, lastBlockHeight, err = persistence.WriteNextBlock(block)
		require.NoError(t, err)
		require.EqualValues(t, wroteBlock, true, "expected write to succeed")
		require.EqualValues(t, lastBlockHeight, block.TransactionsBlock.Header.BlockHeight(), "expected last block height to match written block height to match")
	}
	block = blocks[5]
	wroteBlock, lastBlockHeight, _ = persistence.WriteNextBlock(block)
	require.EqualValues(t, wroteBlock, false, "expected write logic protection to prevent writing on already existing block height")
	require.EqualValues(t, lastBlockHeight, primitives.BlockHeight(10), "expected sequential top block height to pertain")

	for i := 98; i >= 40; i-- {
		block = blocks[i]
		wroteBlock, lastBlockHeight, err = persistence.WriteNextBlock(block)
		require.NoError(t, err)
		require.EqualValues(t, wroteBlock, true, "expected write to succeed")
		require.EqualValues(t, lastBlockHeight, primitives.BlockHeight(10), "expected last block height to match written sequential top block height")
	}

	block = blocks[20]
	wroteBlock, lastBlockHeight, _ = persistence.WriteNextBlock(block)
	require.EqualValues(t, wroteBlock, false, "expected write logic protection to prevent writing not according to last written progress")
	require.EqualValues(t, lastBlockHeight, primitives.BlockHeight(10), "expected sequential top block height to pertain")

	for i := 39; i >= 10; i-- {
		block = blocks[i]
		wroteBlock, lastBlockHeight, err = persistence.WriteNextBlock(block)
		require.NoError(t, err)
		require.EqualValues(t, wroteBlock, true, "expected write to succeed")
	}
	require.EqualValues(t, lastBlockHeight, primitives.BlockHeight(99), "expected last block height to close gap and reach top height")

	block = blocks[99]
	wroteBlock, lastBlockHeight, _ = persistence.WriteNextBlock(block)
	require.NoError(t, err)
	require.EqualValues(t, wroteBlock, true, "expected write to succeed")
	require.EqualValues(t, lastBlockHeight, primitives.BlockHeight(100), "expected sequential top block height to progress")
}

func TestBlockPersistence_WriteLogic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Integration tests in short mode")
	}
	with.Logging(t, func(harness *with.LoggingHarness) {
		ctrlRand := rand.NewControlledRand(t)
		numBlocks := int32(100)
		blocks := builders.RandomizedBlockChain(numBlocks, ctrlRand)

		conf := newTempFileConfig()
		defer conf.cleanDir()

		fsa, closeAdapter, err := NewFilesystemAdapterDriver(harness.Logger, conf)
		require.NoError(t, err)
		defer closeAdapter()
		testBlockPersistenceWriteLogicWithAdapter(t, fsa, blocks)

		fsa = memory.NewBlockPersistence(harness.Logger, metric.NewRegistry())
		testBlockPersistenceWriteLogicWithAdapter(t, fsa, blocks)
	})
}

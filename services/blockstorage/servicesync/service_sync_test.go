// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package servicesync

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSyncLoop(t *testing.T) {
	test.WithContext(func(ctx context.Context) {

		// Set up block source mock
		sourceMock := newBlockSourceMock(4)
		sourceMock.When("GetLastBlock").Times(1)
		sourceMock.When("ScanBlocks", mock.Any, mock.Any, mock.Any).Times(1)

		// Set up target mock
		committerMock := &blockPairCommitterMock{}
		committerHeight := primitives.BlockHeight(0)
		committerMock.When("commitBlockPair", mock.Any, mock.Any).Call(func(ctx context.Context, committedBlockPair *protocol.BlockPairContainer) (primitives.BlockHeight, error) {
			if committedBlockPair.TransactionsBlock.Header.BlockHeight() == committerHeight+1 {
				committerHeight++
			}
			return committerHeight + 1, nil
		}).Times(5)

		// run sync loop
		syncedHeight, err := syncToTopBlock(ctx, sourceMock, committerMock, log.DefaultTestingLogger(t))
		require.NoError(t, err, "expected syncToTopBlock to execute without error")
		require.EqualValues(t, 4, committerHeight, "expected syncToTopBlock to advance committer to source height")
		require.True(t, committerHeight == syncedHeight, "expected syncToTopBlock to return the current block height")

		_, err = sourceMock.Verify()
		require.NoError(t, err)

		_, err = committerMock.Verify()
		require.NoError(t, err)
	})
}

func TestSyncInitialState(t *testing.T) {

	test.WithContext(func(ctx context.Context) {
		// Set up block source mock
		logger := log.DefaultTestingLogger(t)

		sourceTracker := synchronization.NewBlockTracker(logger, 3, 10)
		sourceMock := newBlockSourceMock(3)
		sourceMock.When("GetLastBlock").Times(2)
		sourceMock.When("GetBlockTracker").Return(sourceTracker, nil).AtLeast(0)
		sourceMock.When("ScanBlocks", mock.Any, mock.Any, mock.Any).AtLeast(0)

		// Set up target mock
		committerMock := &blockPairCommitterMock{}
		targetCurrentHeight := primitives.BlockHeight(0)
		targetTracker := synchronization.NewBlockTracker(logger, 0, 10)
		committerMock.When("commitBlockPair", mock.Any, mock.Any).Call(func(ctx context.Context, committedBlockPair *protocol.BlockPairContainer) (primitives.BlockHeight, error) {
			if committedBlockPair.TransactionsBlock.Header.BlockHeight() == targetCurrentHeight+1 {
				targetCurrentHeight++
				targetTracker.IncrementTo(targetCurrentHeight)
			}
			return targetCurrentHeight + 1, nil
		}).Times(5)

		NewServiceBlockSync(ctx, logger, sourceMock, committerMock)

		// Wait for first sync
		err := targetTracker.WaitForBlock(ctx, 3)
		require.NoError(t, err, "expected block committer to be synced to block height 3")

		// push another block
		sourceMock.setLastBlockHeight(4)
		sourceTracker.IncrementTo(4)

		// Wait for second sync
		err = targetTracker.WaitForBlock(ctx, 4)
		require.NoError(t, err, "expected block committer to be synced to block height 4")
		require.EqualValues(t, 4, targetCurrentHeight, "expected block committer to be synced to block height 4")

		_, err = sourceMock.Verify()
		require.NoError(t, err, "blockSource object should be called as expected")

		_, err = committerMock.Verify()
		require.NoError(t, err, "blockPairCommitter should be called as expected")
	})
}

type blockSourceMock struct {
	mock.Mock
	lastBlock *protocol.BlockPairContainer
}

func newBlockSourceMock(height primitives.BlockHeight) *blockSourceMock {
	res := &blockSourceMock{}
	res.setLastBlockHeight(height)
	return res

}

func (bsf *blockSourceMock) setLastBlockHeight(height primitives.BlockHeight) {
	bsf.lastBlock = &protocol.BlockPairContainer{
		TransactionsBlock: &protocol.TransactionsBlockContainer{
			Header: (&protocol.TransactionsBlockHeaderBuilder{BlockHeight: height}).Build(),
		},
		ResultsBlock: &protocol.ResultsBlockContainer{
			Header: (&protocol.ResultsBlockHeaderBuilder{BlockHeight: height}).Build(),
		},
	}
}

// TODO V1 - this duplicates logic form inMemoryBlockPersistence. Do we really need a mock object here?
func (bsf *blockSourceMock) ScanBlocks(from primitives.BlockHeight, pageSize uint8, f adapter.CursorFunc) error {
	bsf.Called(from, pageSize, f)

	topBlockHeight := bsf.lastBlock.ResultsBlock.Header.BlockHeight()

	// build a dummy block-chain
	blocks := make([]*protocol.BlockPairContainer, topBlockHeight)
	for i := range blocks {
		blocks[i] = &protocol.BlockPairContainer{
			TransactionsBlock: &protocol.TransactionsBlockContainer{Header: (&protocol.TransactionsBlockHeaderBuilder{BlockHeight: from + primitives.BlockHeight(i)}).Build()},
			ResultsBlock:      &protocol.ResultsBlockContainer{Header: (&protocol.ResultsBlockHeaderBuilder{BlockHeight: from + primitives.BlockHeight(i)}).Build()},
		}
	}

	// invoke f repeatedly with pages
	cont := true
	for from <= topBlockHeight && cont {
		topPageIndex := int(from) + int(pageSize) - 1
		if topPageIndex > len(blocks) {
			topPageIndex = len(blocks)
		}
		cont = f(from, blocks[from-1:topPageIndex])
		from = primitives.BlockHeight(topPageIndex) + 1
	}

	return nil
}

func (bsf *blockSourceMock) GetBlockTracker() *synchronization.BlockTracker {
	return bsf.Called().Get(0).(*synchronization.BlockTracker)
}
func (bsf *blockSourceMock) GetLastBlock() (*protocol.BlockPairContainer, error) {
	bsf.Called()
	return bsf.lastBlock, nil
}

type blockPairCommitterMock struct {
	mock.Mock
}

func (stm *blockPairCommitterMock) commitBlockPair(ctx context.Context, committedBlockPair *protocol.BlockPairContainer) (primitives.BlockHeight, error) {
	ret := stm.Mock.Called(ctx, committedBlockPair)
	return ret.Get(0).(primitives.BlockHeight), ret.Error(1)
}

func (stm *blockPairCommitterMock) getServiceName() string {
	return "mock-committer"
}

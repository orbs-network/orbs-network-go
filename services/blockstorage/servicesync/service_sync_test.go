package servicesync

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
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
		sourceMock.When("GetBlocks", mock.Any, mock.Any).Times(5)

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
		syncedHeight, err := syncOnce(ctx, sourceMock, committerMock, log.GetLogger())
		require.NoError(t, err, "expected syncOnce to execute without error")
		require.EqualValues(t, 4, committerHeight, "expected syncOnce to advance committer to source height")
		require.True(t, committerHeight == syncedHeight, "expected syncOnce to return the current block height")

		_, err = sourceMock.Verify()
		require.NoError(t, err)

		_, err = committerMock.Verify()
		require.NoError(t, err)
	})
}

func TestSyncInitialState(t *testing.T) {

	test.WithContext(func(ctx context.Context) {
		// Set up block source mock
		sourceTracker := synchronization.NewBlockTracker(3, 10)
		sourceMock := newBlockSourceMock(3)
		sourceMock.When("GetLastBlock").Times(2)
		sourceMock.When("GetBlockTracker").Return(sourceTracker, nil).AtLeast(0)
		sourceMock.When("GetBlocks", mock.Any, mock.Any).Times(5)

		// Set up target mock
		committerMock := &blockPairCommitterMock{}
		targetCurrentHeight := primitives.BlockHeight(0)
		targetTracker := synchronization.NewBlockTracker(0, 10)
		committerMock.When("commitBlockPair", mock.Any, mock.Any).Call(func(ctx context.Context, committedBlockPair *protocol.BlockPairContainer) (primitives.BlockHeight, error) {
			if committedBlockPair.TransactionsBlock.Header.BlockHeight() == targetCurrentHeight+1 {
				targetTracker.IncrementHeight()
				targetCurrentHeight++
			}
			return targetCurrentHeight + 1, nil
		}).Times(5)

		NewServiceBlockSync(ctx, log.GetLogger(), sourceMock, committerMock)

		// Wait for first sync
		err := targetTracker.WaitForBlock(ctx, 3)
		require.NoError(t, err, "expected block committer to be synced to block height 3")

		// push another block
		sourceMock.setLastBlockHeight(4)
		sourceTracker.IncrementHeight()

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

// TODO(https://github.com/orbs-network/orbs-network-go/issues/582) - this fake implementation assumes there is no genesis block, Fix once addressing genesis
func (bsf *blockSourceMock) GetBlocks(first primitives.BlockHeight, last primitives.BlockHeight) (blocks []*protocol.BlockPairContainer, firstReturnedBlockHeight primitives.BlockHeight, lastReturnedBlockHeight primitives.BlockHeight, err error) {
	bsf.Called(first, last)
	result := make([]*protocol.BlockPairContainer, last-first)
	for i := range result {
		result[i] = &protocol.BlockPairContainer{
			TransactionsBlock: &protocol.TransactionsBlockContainer{Header: (&protocol.TransactionsBlockHeaderBuilder{BlockHeight: first + primitives.BlockHeight(i)}).Build()},
			ResultsBlock:      &protocol.ResultsBlockContainer{Header: (&protocol.ResultsBlockHeaderBuilder{BlockHeight: first + primitives.BlockHeight(i)}).Build()},
		}
	}
	return result, first, last, nil
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

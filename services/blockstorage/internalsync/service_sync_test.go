package internalsync

import (
	"context"
	"github.com/orbs-network/go-mock"
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
		sourceMock := &blockSourceMock{}
		sourceMock.When("GetNumBlocks").Return(primitives.BlockHeight(2), nil).Times(1)
		sourceMock.When("GetBlocks", mock.Any, mock.Any).Call(func(first primitives.BlockHeight, last primitives.BlockHeight) (blocks []*protocol.BlockPairContainer, firstReturnedBlockHeight primitives.BlockHeight, lastReturnedBlockHeight primitives.BlockHeight, err error) {
			result := make ([]*protocol.BlockPairContainer, last - first + 1)
			for i := range result {
				result[i] = &protocol.BlockPairContainer{
					TransactionsBlock: &protocol.TransactionsBlockContainer{Header: (&protocol.TransactionsBlockHeaderBuilder{BlockHeight: first + primitives.BlockHeight(i)}).Build()},
				}
			}
			return result, first, last, nil
		}).Times(3)

		// Set up target mock
		targetMock := &syncTargetMock{}
		currentHeight := primitives.BlockHeight(0)
		targetMock.When("callback", mock.Any, mock.Any).Call(func(ctx context.Context, committedBlockPair *protocol.BlockPairContainer) (primitives.BlockHeight, error) {
			if committedBlockPair.TransactionsBlock.Header.BlockHeight() == currentHeight + 1 {
				currentHeight++
			}
			return currentHeight + 1, nil
		}).Times(3)

		// run sync loop
		reportedHeight, err := syncOnce(ctx, sourceMock, targetMock.callback)
		require.NoError(t, err)
		require.True(t, currentHeight == reportedHeight)

		_, err = sourceMock.Verify()
		require.NoError(t, err)

		_, err = targetMock.Verify()
		require.NoError(t, err)

		require.EqualValues(t, 2, currentHeight)
	})
}

func TestSyncInitialState(t *testing.T) {

	test.WithContext(func(ctx context.Context) {
		// Set up block source mock
		sourceTracker := synchronization.NewBlockTracker(2, 10)
		sourceCurrentHeight := primitives.BlockHeight(2)
		sourceMock := &blockSourceMock{}
		sourceMock.When("GetNumBlocks").Call(func () (primitives.BlockHeight, error) {return sourceCurrentHeight, nil}).Times(2)
		sourceMock.When("GetBlockTracker").Return(sourceTracker, nil).AtLeast(0)
		sourceMock.When("GetBlocks", mock.Any, mock.Any).Call(func(first primitives.BlockHeight, last primitives.BlockHeight) (blocks []*protocol.BlockPairContainer, firstReturnedBlockHeight primitives.BlockHeight, lastReturnedBlockHeight primitives.BlockHeight, err error) {
			result := make ([]*protocol.BlockPairContainer, last - first + 1)
			for i := range result {
				result[i] = &protocol.BlockPairContainer{
					TransactionsBlock: &protocol.TransactionsBlockContainer{Header: (&protocol.TransactionsBlockHeaderBuilder{BlockHeight: first + primitives.BlockHeight(i)}).Build()},
				}
			}
			return result, first, last, nil
		}).Times(4)

		// Set up target mock
		targetMock := &syncTargetMock{}
		targetCurrentHeight := primitives.BlockHeight(0)
		targetTracker := synchronization.NewBlockTracker(0, 10)
		targetMock.When("callback", mock.Any, mock.Any).Call(func(ctx context.Context, committedBlockPair *protocol.BlockPairContainer) (primitives.BlockHeight, error) {
			if committedBlockPair.TransactionsBlock.Header.BlockHeight() == targetCurrentHeight+ 1 {
				targetTracker.IncrementHeight()
				targetCurrentHeight++
			}
			return targetCurrentHeight + 1, nil
		}).Times(4)

		StartSupervised(ctx, nil, sourceMock, targetMock.callback)

		// Wait for first sync
		err := targetTracker.WaitForBlock(ctx, 2)
		require.NoError(t, err)

		// push another block
		sourceCurrentHeight++
		sourceTracker.IncrementHeight()

		// Wait for second sync
		err = targetTracker.WaitForBlock(ctx, 3)
		require.NoError(t, err)

		_, err = sourceMock.Verify()
		require.NoError(t, err)

		_, err = targetMock.Verify()
		require.NoError(t, err)

		require.EqualValues(t, 3, targetCurrentHeight)
	})
}

type blockSourceMock struct {
	mock.Mock
}
func (bsf *blockSourceMock) GetBlocks(first primitives.BlockHeight, last primitives.BlockHeight) (blocks []*protocol.BlockPairContainer, firstReturnedBlockHeight primitives.BlockHeight, lastReturnedBlockHeight primitives.BlockHeight, err error) {
	ret := bsf.Called(first, last)
	return ret.Get(0).([]*protocol.BlockPairContainer), ret.Get(1).(primitives.BlockHeight), ret.Get(2).(primitives.BlockHeight), ret.Error(3)
}

func (bsf *blockSourceMock) GetBlockTracker() *synchronization.BlockTracker {
	return bsf.Called().Get(0).(*synchronization.BlockTracker)
}
func (bsf *blockSourceMock) GetNumBlocks() (primitives.BlockHeight, error) {
	ret := bsf.Called()
	return ret.Get(0).(primitives.BlockHeight), ret.Error(1)
}

type syncTargetMock struct {
	mock.Mock
}

func (stm *syncTargetMock) callback(ctx context.Context, committedBlockPair *protocol.BlockPairContainer) (primitives.BlockHeight, error) {
	ret := stm.Mock.Called(ctx, committedBlockPair)
	return ret.Get(0).(primitives.BlockHeight), ret.Error(1)
}
package synchronization

import (
	"github.com/stretchr/testify/require"
	"sync/atomic"
	"testing"
	"time"
)

func TestWaitForBlockOutsideOfGraceFailsImmediately(t *testing.T) {
	tracker := NewBlockTracker(1, 1, 10*time.Millisecond)

	err := tracker.WaitForBlock(3)
	require.EqualError(t, err, "requested future block outside of grace range", "did not fail immediately")
}

func TestWaitForBlockWithinGraceFailsAfterTimeout(t *testing.T) {

	tracker := NewBlockTracker(1, 1, 1*time.Millisecond)
	err := tracker.WaitForBlock(2)
	require.EqualError(t, err, "timed out waiting for block at height 2", "did not timeout as expected")
}

func TestWaitForBlockWithinGraceDealsWithIntegerUnderflow(t *testing.T) {

	tracker := NewBlockTracker(0, 5, 1*time.Millisecond)
	err := tracker.WaitForBlock(2)
	require.EqualError(t, err, "timed out waiting for block at height 2", "did not timeout as expected")
}

func TestWaitForBlockWithinGraceReturnsWhenBlockHeightReachedBeforeTimeoutAfterWaiting(t *testing.T) {
	tracker := NewBlockTracker(1, 2, 1*time.Second)

	var waitCount int32
	internalWaitChan := make(chan int32)
	tracker.fireOnWait = func() {
		internalWaitChan <- atomic.AddInt32(&waitCount, 1)
	}

	doneWait := make(chan error)
	go func() {
		doneWait <- tracker.WaitForBlock(3)
	}()

	require.EqualValues(t, 1, <-internalWaitChan, "did not block before the first increment")
	tracker.IncrementHeight()
	require.EqualValues(t, 2, <-internalWaitChan, "did not block before the second increment")
	tracker.IncrementHeight()

	require.NoError(t, <-doneWait, "did not return as expected")
}

func TestWaitForBlockWithinGraceSupportsTwoConcurrentWaiters(t *testing.T) {
	tracker := NewBlockTracker(1, 1, 1*time.Second)

	var waitCount int32
	internalWaitChan := make(chan int32)
	tracker.fireOnWait = func() {
		internalWaitChan <- atomic.AddInt32(&waitCount, 1)
	}

	doneWait := make(chan error)
	waiter := func() {
		doneWait <- tracker.WaitForBlock(2)
	}
	go waiter()
	go waiter()

	selectIterationsBeforeIncrement := <-internalWaitChan
	require.EqualValues(t, 1, selectIterationsBeforeIncrement, "did not enter select before returning")
	selectIterationsBeforeIncrement = <-internalWaitChan
	require.EqualValues(t, 2, selectIterationsBeforeIncrement, "did not enter select before returning")

	tracker.IncrementHeight()

	require.NoError(t, <-doneWait, "first waiter did not return as expected")
	require.NoError(t, <-doneWait, "second waiter did not return as expected")
}

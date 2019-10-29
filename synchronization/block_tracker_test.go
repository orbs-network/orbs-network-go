// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package synchronization

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/wait"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/stretchr/testify/require"
	"sync/atomic"
	"testing"
)

func TestWaitForBlockOutsideOfGraceFailsImmediately(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			tracker := NewBlockTracker(parent.Logger, 1, 1)

			err := tracker.WaitForBlock(ctx, 3)
			require.EqualError(t, err, "requested future block outside of grace range", "did not fail immediately")
		})
	})
}

func TestWaitForBlockWithinGraceFailsWhenContextEnds(t *testing.T) {
	with.Context(func(parentCtx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			ctx, cancel := context.WithCancel(parentCtx)
			tracker := NewBlockTracker(parent.Logger, 1, 1)
			cancel()
			err := tracker.WaitForBlock(ctx, 2)
			require.EqualError(t, err, "aborted while waiting for block at height 2: context canceled", "did not fail as expected")
		})
	})
}

func TestWaitForBlockWithinGraceDealsWithIntegerUnderflow(t *testing.T) {
	with.Context(func(parentCtx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			ctx, cancel := context.WithCancel(parentCtx)
			tracker := NewBlockTracker(parent.Logger, 0, 5)
			cancel()
			err := tracker.WaitForBlock(ctx, 2)
			require.EqualError(t, err, "aborted while waiting for block at height 2: context canceled", "did not fail as expected")
		})
	})
}

func TestWaitForBlockWithinGraceReturnsWhenBlockHeightReachedBeforeContextEnds(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			tracker := NewBlockTracker(parent.Logger, 1, 2)

			var waitCount int32
			internalWaitChan := make(chan int32)
			tracker.fireOnWait = func() {
				internalWaitChan <- atomic.AddInt32(&waitCount, 1)
			}

			doneWait := make(chan error)
			go func() {
				doneWait <- tracker.WaitForBlock(ctx, 3)
			}()

			require.EqualValues(t, 1, <-internalWaitChan, "did not block before the first increment")
			require.NotPanics(t, func() {
				tracker.IncrementTo(2)
			})
			require.EqualValues(t, 2, <-internalWaitChan, "did not block before the second increment")
			require.NotPanics(t, func() {
				tracker.IncrementTo(3)
			})

			require.NoError(t, <-doneWait, "did not return as expected")
		})
	})
}

func TestWaitForBlockWithinGraceSupportsTwoConcurrentWaiters(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			tracker := NewBlockTracker(parent.Logger, 1, 1)

			internalWaitChan := make(chan struct{})
			tracker.fireOnWait = func() {
				internalWaitChan <- struct{}{}
			}

			doneWait := make(chan error)
			waiter := func() {
				doneWait <- tracker.WaitForBlock(ctx, 2)
			}
			go waiter()
			go waiter()

			require.NoError(t, wait.ForSignal(internalWaitChan), "did not enter select before returning")
			require.NoError(t, wait.ForSignal(internalWaitChan), "did not enter select before returning")

			require.NotPanics(t, func() {
				tracker.IncrementTo(2)
			})

			require.NoError(t, <-doneWait, "first waiter did not return as expected")
			require.NoError(t, <-doneWait, "second waiter did not return as expected")
		})
	})
}

func TestBlockTracker_ReachedHeight_RejectsWrongHeight(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			tracker := NewBlockTracker(parent.Logger, 1, 1)

			require.Panics(t, func() {
				tracker.IncrementTo(3)
			}, "should have rejected non-sequential height")
		})
	})
}

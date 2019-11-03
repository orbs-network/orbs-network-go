// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package synchronization_test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"sync/atomic"
	"testing"
	"time"
)

type nopLogger struct{}

func (c *nopLogger) Error(message string, fields ...*log.Field) {}

func newNopLogger() *nopLogger {
	return &nopLogger{}
}

func TestPeriodicalTriggerStartsOk(t *testing.T) {
	logger := newNopLogger()
	var handlerInvocationCount uint32
	trigger := func() {
		atomic.AddUint32(&handlerInvocationCount, 1)
	}
	tickTime := time.Microsecond
	p := synchronization.NewPeriodicalTrigger(context.Background(), "a periodical trigger", tickTime, logger, trigger, nil)

	requireAtomicGreaterThan(t, &handlerInvocationCount, 0, "expected trigger to have ticked once promptly")

	p.Stop()
}

func TestPeriodicalTrigger_Stop(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		x := 0
		p := synchronization.NewPeriodicalTrigger(context.Background(), "a periodical trigger", time.Millisecond*2, harness.Logger, func() { x++ }, nil)
		harness.Supervise(p)
		p.Stop()
		time.Sleep(3 * time.Millisecond)
		require.Equal(t, 0, x, "expected no ticks")
	})
}

func TestPeriodicalTrigger_StopAfterTrigger(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		x := 0
		p := synchronization.NewPeriodicalTrigger(context.Background(), "a periodical trigger", time.Millisecond, harness.Logger, func() { x++ }, nil)
		harness.Supervise(p)
		time.Sleep(time.Microsecond * 1100)
		p.Stop()
		xValueOnStop := x
		time.Sleep(time.Millisecond * 5)
		require.Equal(t, xValueOnStop, x, "expected one tick due to stop")

	})
}

func TestPeriodicalTriggerStopOnContextCancel(t *testing.T) {
	with.Concurrency(t, func(parent context.Context, harness *with.ConcurrencyHarness) {
		ctx, cancel := context.WithCancel(parent)
		x := 0
		p := synchronization.NewPeriodicalTrigger(ctx, "a periodical trigger", time.Millisecond*2, harness.Logger, func() { x++ }, nil)
		harness.Supervise(p)
		cancel()
		time.Sleep(3 * time.Millisecond)
		require.Equal(t, 0, x, "expected no ticks")
	})
}

func TestPeriodicalTriggerStopWorksWhenContextIsCancelled(t *testing.T) {
	with.Concurrency(t, func(parent context.Context, harness *with.ConcurrencyHarness) {
		ctx, cancel := context.WithCancel(parent)
		cancel() // send a cancelled context to reduce chances of trigger being called even once
		x := 0
		p := synchronization.NewPeriodicalTrigger(ctx, "a periodical trigger", time.Millisecond*2, harness.Logger, func() { x++ }, nil)
		harness.Supervise(p)
		time.Sleep(3 * time.Millisecond)
		require.Equal(t, 0, x, "expected no ticks")
		p.Stop()
		require.Equal(t, 0, x, "expected stop to not block")

	})
}

func TestPeriodicalTriggerStopOnContextCancelWithStopAction(t *testing.T) {
	with.Concurrency(t, func(parent context.Context, harness *with.ConcurrencyHarness) {
		ctx, cancel := context.WithCancel(parent)
		ch := make(chan struct{})
		p := synchronization.NewPeriodicalTrigger(ctx, "a periodical trigger", time.Millisecond*2, harness.Logger, func() {}, func() { close(ch) })
		harness.Supervise(p)
		cancel()
		_, ok := <-ch
		require.False(t, ok, "expected trigger stop action to close the channel")

	})
}

func TestPeriodicalTriggerRunsOnStopAction(t *testing.T) {
	with.Concurrency(t, func(parent context.Context, harness *with.ConcurrencyHarness) {
		latch := make(chan struct{})
		x := 0
		p := synchronization.NewPeriodicalTrigger(context.Background(), "a periodical trigger", time.Second, harness.Logger, func() { x++ }, func() {
			x = 20
			latch <- struct{}{}
		})
		harness.Supervise(p)
		p.Stop()
		<-latch // wait for stop to happen...
		require.Equal(t, 20, x, "expected x to have the stop value")

	})
}

func TestPeriodicalTriggerKeepsGoingOnPanic(t *testing.T) {
	with.Concurrency(t, func(parent context.Context, harness *with.ConcurrencyHarness) {

		logger := newNopLogger()
		var handlerInvocationCount uint32
		p := synchronization.NewPeriodicalTrigger(context.Background(), "a periodical trigger", time.Millisecond, logger, func() {
			atomic.AddUint32(&handlerInvocationCount, 1)
			panic("we should not see this other than the logs")
		}, nil)

		requireAtomicGreaterThan(t, &handlerInvocationCount, 1, "expected trigger to have ticked more than once even though it panics")

		p.Stop()
	})

}

func requireAtomicGreaterThan(t *testing.T, counterPtr *uint32, val uint32, msgAndArgs ...interface{}) {
	require.True(t, test.Eventually(1*time.Second, func() bool {
		return atomic.LoadUint32(counterPtr) > val
	}), msgAndArgs...)
}

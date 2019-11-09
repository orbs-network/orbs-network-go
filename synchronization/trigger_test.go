// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package synchronization_test

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/synchronization"
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

// simple, thread-safe testing tool
type TestFlag struct {
	t     testing.TB
	value uint32
}

func NewTestFlag(t testing.TB) *TestFlag {
	return &TestFlag{
		t:     t,
		value: 0,
	}
}

func (c *TestFlag) Toggle() {
	atomic.StoreUint32(&(c.value), 1)
}

func (c *TestFlag) EventuallyToggled(message string) {
	require.Eventually(c.t, func() bool { return atomic.CompareAndSwapUint32(&(c.value), 1, 0) }, time.Second, time.Millisecond, message)
}

func (c *TestFlag) NeverToggled(message string) {
	time.Sleep(time.Millisecond * 500)
	require.True(c.t, atomic.LoadUint32(&(c.value)) == 0, message)
}

// Ticker for testing.
// Only sends ticks when .Tick() is called explicitly
// has stop TestFlag to easily assert calls to .Stop()
type TestTicker struct {
	c    chan time.Time
	stop TestFlag
}

func (t *TestTicker) C() <-chan time.Time {
	return t.c
}

func (t *TestTicker) Tick() {
	t.c <- time.Now()
}

func (t *TestTicker) Stop() {
	t.stop.Toggle()
}

func NewTestTicker(t *testing.T) *TestTicker {
	return &TestTicker{
		c:    make(chan time.Time),
		stop: *NewTestFlag(t),
	}
}

func TestPeriodicalTriggerStartsOk(t *testing.T) {
	logger := newNopLogger()
	triggerFlag := NewTestFlag(t)
	ticker := synchronization.NewTimeTicker(time.Millisecond)
	p := synchronization.NewPeriodicalTrigger(context.Background(), "a periodical trigger", ticker, logger, triggerFlag.Toggle, nil)
	defer p.Stop()
	triggerFlag.EventuallyToggled("expected trigger to have ticked once immediately")
}

func TestPeriodicalTriggerNoTicksNoTriggers(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		ticker := NewTestTicker(t) // does not tick by default
		triggerFlag := NewTestFlag(t)
		p := synchronization.NewPeriodicalTrigger(ctx, "a periodical trigger", ticker, harness.Logger, triggerFlag.Toggle, nil)
		defer p.Stop()
		harness.Supervise(p)
		triggerFlag.NeverToggled("expected no triggers when no ticks")
	})
}

func TestPeriodicalTriggerTriggersOnTicks(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		ticker := NewTestTicker(t)
		triggerFlag := NewTestFlag(t)
		p := synchronization.NewPeriodicalTrigger(ctx, "a periodical trigger", ticker, harness.Logger, triggerFlag.Toggle, nil)
		defer p.Stop()
		harness.Supervise(p)
		for i := 0; i < 5; i++ {
			ticker.Tick()
			triggerFlag.EventuallyToggled(fmt.Sprintf("expected %d trigger invocations", i))
		}
	})
}

func TestPeriodicalTriggerKeepsTriggeringAfterPanic(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		logger := newNopLogger()
		ticker := NewTestTicker(t)
		triggerFlag := NewTestFlag(t)
		p := synchronization.NewPeriodicalTrigger(ctx, "a periodical trigger", ticker, logger, func() {
			triggerFlag.Toggle()
			panic("we should not see this other than the logs")
		}, nil)
		defer p.Stop()
		harness.Supervise(p)
		for i := 0; i < 5; i++ {
			ticker.Tick()
			triggerFlag.EventuallyToggled(fmt.Sprintf("expected %d trigger invocations even though it panics", i))
		}
	})
}

func TestPeriodicalTriggerStopsTickerOnCallToStop(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		ticker := NewTestTicker(t)
		p := synchronization.NewPeriodicalTrigger(ctx, "a periodical trigger", ticker, harness.Logger, func() {}, nil)
		harness.Supervise(p)
		p.Stop()
		ticker.stop.EventuallyToggled("expected ticker.Stop() to have been called")
	})
}

func TestPeriodicalTriggerStopsTickerOnContextCancel(t *testing.T) {
	with.Concurrency(t, func(parent context.Context, harness *with.ConcurrencyHarness) {
		ctx, cancel := context.WithCancel(parent)
		ticker := NewTestTicker(t)
		p := synchronization.NewPeriodicalTrigger(ctx, "a periodical trigger", ticker, harness.Logger, func() {}, nil)
		defer p.Stop()
		harness.Supervise(p)
		cancel()
		ticker.stop.EventuallyToggled("expected ticker.Stop() to have been called")
	})
}

func TestPeriodicalTriggerCallsStopHookOnCallToStop(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		ticker := NewTestTicker(t)
		onStop := NewTestFlag(t)
		p := synchronization.NewPeriodicalTrigger(ctx, "a periodical trigger", ticker, harness.Logger, func() {}, onStop.Toggle)
		harness.Supervise(p)
		p.Stop()
		onStop.EventuallyToggled("expected onStop() to have been called")
		ticker.stop.EventuallyToggled("expected ticker.Stop() to have been called")
	})
}

func TestPeriodicalTriggerCallsStopHookOnContextCancel(t *testing.T) {
	with.Concurrency(t, func(parent context.Context, harness *with.ConcurrencyHarness) {
		ctx, cancel := context.WithCancel(parent)
		ticker := NewTestTicker(t)
		onStop := NewTestFlag(t)
		p := synchronization.NewPeriodicalTrigger(ctx, "a periodical trigger", ticker, harness.Logger, func() {}, onStop.Toggle)
		defer p.Stop()
		harness.Supervise(p)
		cancel()
		onStop.EventuallyToggled("expected onStop() to have been called")
		ticker.stop.EventuallyToggled("expected ticker.Stop() to have been called")
	})
}

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
	count uint32
}

func NewTestFlag(t testing.TB) *TestFlag {
	return &TestFlag{
		t:     t,
		count: 0,
	}
}

func (c *TestFlag) Toggle() {
	atomic.StoreUint32(&(c.count), 1)
}
func (c *TestFlag) EventuallyToggled(message string) {
	require.Eventually(c.t, func() bool { return atomic.CompareAndSwapUint32(&(c.count), 1, 0) }, time.Second, time.Millisecond, message)
}

func (c *TestFlag) NotEventuallyToggled(message string) {
	time.Sleep(time.Millisecond * 500)
	require.True(c.t, atomic.LoadUint32(&(c.count)) == 0, message)
}

func TestPeriodicalTriggerStartsOk(t *testing.T) {
	logger := newNopLogger()
	triggerFlag := NewTestFlag(t)
	ticker := synchronization.NewTimeTicker(time.Millisecond)
	p := synchronization.NewPeriodicalTrigger(context.Background(), "a periodical trigger", ticker, logger, func() { triggerFlag.Toggle() }, nil)
	defer p.Stop()
	triggerFlag.EventuallyToggled("expected trigger to have ticked once immediately")
}

func TestPeriodicalTriggerNoTicksNoTriggers(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		ticker := synchronization.NewHookTicker() // does not tick by default
		triggerFlag := NewTestFlag(t)
		p := synchronization.NewPeriodicalTrigger(context.Background(), "a periodical trigger", ticker, harness.Logger, func() { triggerFlag.Toggle() }, nil)
		defer p.Stop()
		harness.Supervise(p)
		triggerFlag.NotEventuallyToggled("expected no triggers when no ticks")
	})
}

func TestPeriodicalTriggerTriggersOnTicks(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		ticker := synchronization.NewHookTicker()
		ch := make(chan time.Time)
		ticker.C_ = func() <-chan time.Time { return ch }
		triggerFlag := NewTestFlag(t)
		p := synchronization.NewPeriodicalTrigger(context.Background(), "a periodical trigger", ticker, harness.Logger, func() { triggerFlag.Toggle() }, nil)
		defer p.Stop()
		harness.Supervise(p)
		for i := 0; i < 5; i++ {
			ch <- time.Now()
			triggerFlag.EventuallyToggled(fmt.Sprintf("expected %d trigger invocations", i))
		}
	})
}

func TestPeriodicalTriggerKeepsTriggeringAfterPanic(t *testing.T) {
	with.Concurrency(t, func(parent context.Context, harness *with.ConcurrencyHarness) {
		logger := newNopLogger()
		ticker := synchronization.NewHookTicker()
		ch := make(chan time.Time)
		ticker.C_ = func() <-chan time.Time { return ch }
		triggerFlag := NewTestFlag(t)
		p := synchronization.NewPeriodicalTrigger(context.Background(), "a periodical trigger", ticker, logger, func() {
			triggerFlag.Toggle()
			panic("we should not see this other than the logs")
		}, nil)
		defer p.Stop()
		harness.Supervise(p)
		for i := 0; i < 5; i++ {
			ch <- time.Now()
			triggerFlag.EventuallyToggled(fmt.Sprintf("expected %d trigger invocations even though it panics", i))
		}
	})
}

func TestPeriodicalTriggerStopsTickerOnCallToStop(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		ticker := synchronization.NewHookTicker()
		wasStopped := NewTestFlag(t)
		ticker.Stop_ = func() { wasStopped.Toggle() }
		p := synchronization.NewPeriodicalTrigger(context.Background(), "a periodical trigger", ticker, harness.Logger, func() {}, nil)
		harness.Supervise(p)
		p.Stop()
		wasStopped.EventuallyToggled("expected ticker.Stop() to have been called")
	})
}

func TestPeriodicalTriggerStopsTickerOnContextCancel(t *testing.T) {
	with.Concurrency(t, func(parent context.Context, harness *with.ConcurrencyHarness) {
		ctx, cancel := context.WithCancel(parent)
		ticker := synchronization.NewHookTicker()
		wasStopped := NewTestFlag(t)
		ticker.Stop_ = func() { wasStopped.Toggle() }
		p := synchronization.NewPeriodicalTrigger(ctx, "a periodical trigger", ticker, harness.Logger, func() {}, nil)
		defer p.Stop()
		harness.Supervise(p)
		cancel()
		wasStopped.EventuallyToggled("expected ticker.Stop() to have been called")
	})
}

func TestPeriodicalTriggerCallsStopHookOnCallToStop(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		ticker := synchronization.NewHookTicker()
		wasStopped := NewTestFlag(t)
		p := synchronization.NewPeriodicalTrigger(context.Background(), "a periodical trigger", ticker, harness.Logger, func() {}, func() { wasStopped.Toggle() })
		harness.Supervise(p)
		p.Stop()
		wasStopped.EventuallyToggled("expected ticker.Stop() to have been called")
	})
}

func TestPeriodicalTriggerCallsStopHookOnContextCancel(t *testing.T) {
	with.Concurrency(t, func(parent context.Context, harness *with.ConcurrencyHarness) {
		ctx, cancel := context.WithCancel(parent)
		ticker := synchronization.NewHookTicker()
		wasStopped := NewTestFlag(t)
		p := synchronization.NewPeriodicalTrigger(ctx, "a periodical trigger", ticker, harness.Logger, func() {}, func() { wasStopped.Toggle() })
		defer p.Stop()
		harness.Supervise(p)
		cancel()
		wasStopped.EventuallyToggled("expected ticker.Stop() to have been called")
	})
}

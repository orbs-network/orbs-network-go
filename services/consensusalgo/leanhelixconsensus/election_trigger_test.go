// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package leanhelixconsensus

import (
	"context"
	lhmetrics "github.com/orbs-network/lean-helix-go/instrumentation/metrics"
	"github.com/orbs-network/lean-helix-go/services/interfaces"
	"github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"runtime/pprof"
	"testing"
	"time"
)

func buildElectionTrigger(ctx context.Context, t *testing.T, timeout time.Duration) interfaces.ElectionTrigger {
	et := NewExponentialBackoffElectionTrigger(log.DefaultTestingLogger(t), timeout, nil)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case trigger := <-et.ElectionChannel():
				trigger(ctx)
			}
		}
	}()

	return et
}

func TestCallbackTrigger(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		et := buildElectionTrigger(ctx, t, time.Millisecond)

		fromTrigger := make(chan struct{})
		cb := func(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, onElectionCB func(m lhmetrics.ElectionMetrics)) {
			fromTrigger <- struct{}{}
		}
		et.RegisterOnElection(ctx, 20, 0, cb)

		<-fromTrigger // test will timeout if it does not trigger
	})
}

func TestCallbackTriggerOnce(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		et := buildElectionTrigger(ctx, t, time.Millisecond)

		callCount := 0
		cb := func(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, onElectionCB func(m lhmetrics.ElectionMetrics)) {
			callCount++
		}
		et.RegisterOnElection(ctx, 10, 0, cb)

		time.Sleep(250 * time.Millisecond)

		require.Exactly(t, 1, callCount, "Trigger callback called more than once")
	})
}

func TestCallbackTriggerTwiceInARow(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		et := buildElectionTrigger(ctx, t, time.Millisecond)

		callCount := 0
		cb := func(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, onElectionCB func(m lhmetrics.ElectionMetrics)) {
			callCount++
		}
		et.RegisterOnElection(ctx, 10, 0, cb)

		time.Sleep(25 * time.Millisecond)

		et.RegisterOnElection(ctx, 11, 0, cb)
		time.Sleep(25 * time.Millisecond)

		require.Exactly(t, 2, callCount, "Trigger callback twice without getting stuck")
	})
}

func TestIgnoreSameViewOrHeight(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		et := buildElectionTrigger(ctx, t, time.Millisecond)

		callCount := 0
		cb := func(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, onElectionCB func(m lhmetrics.ElectionMetrics)) {
			callCount++
		}

		et.RegisterOnElection(ctx, 10, 0, cb)
		time.Sleep(25 * time.Millisecond)
		et.RegisterOnElection(ctx, 10, 0, cb)
		time.Sleep(25 * time.Millisecond)

		require.Exactly(t, 1, callCount, "Trigger callback called more than once")
	})
}

func TestHeightChangeCausesRegister(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		et := buildElectionTrigger(ctx, t, 30*time.Millisecond)

		beforeSecondRegister := false
		cb1 := func(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, onElectionCB func(m lhmetrics.ElectionMetrics)) {
			beforeSecondRegister = true
		}

		afterSecondRegister := false
		cb2 := func(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, onElectionCB func(m lhmetrics.ElectionMetrics)) {
			afterSecondRegister = true
		}

		et.RegisterOnElection(ctx, 10, 0, cb1)
		et.RegisterOnElection(ctx, 11, 0, cb2)
		time.Sleep(60 * time.Millisecond)

		require.False(t, beforeSecondRegister, "should not trigger the first one")
		require.True(t, afterSecondRegister, "should only trigger the second one")

	})
}

func TestViewChangesCausesRegister(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		et := buildElectionTrigger(ctx, t, time.Millisecond)

		wasCalled := false
		cb := func(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, onElectionCB func(m lhmetrics.ElectionMetrics)) {
			wasCalled = true
		}

		et.RegisterOnElection(ctx, 10, 2, cb)  // 2 ** 2 * 1 = 4
		et.RegisterOnElection(ctx, 10, 20, cb) // 2 ** 20 * 1 = 1048576
		time.Sleep(25 * time.Millisecond)

		require.False(t, wasCalled, "Trigger the callback even if a new Register was called with a new view")
	})
}

func TestViewPowTimeout(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		et := buildElectionTrigger(ctx, t, 10*time.Millisecond)

		fromTrigger := make(chan struct{})
		cb := func(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, onElectionCB func(m lhmetrics.ElectionMetrics)) {
			fromTrigger <- struct{}{}
		}

		require.EqualValues(t, 40*time.Millisecond, et.CalcTimeout(2), "view based calculation invalid")
		require.EqualValues(t, 80*time.Millisecond, et.CalcTimeout(3), "view based calculation invalid")

		et.RegisterOnElection(ctx, 10, 2, cb)
		select {
		case <-fromTrigger:
			require.Fail(t, "view timeout calculation not set up correctly in register")
		case <-time.After(10 * time.Millisecond):
		}
	})
}

type testWriter testing.T

func (t *testWriter) Write(p []byte) (n int, err error) {
	t.Log(string(p))
	return 0, nil
}

func TestElectionTriggerDoesNotLeak(t *testing.T) {
	// this test checks that after multiple registrations, there are no goroutine leaks
	writer := testWriter(*t)
	test.WithContext(func(ctx context.Context) {
		et := buildElectionTrigger(ctx, t, time.Microsecond)

		callCount := 0
		cb := func(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, onElectionCB func(m lhmetrics.ElectionMetrics)) {
			callCount++
		}
		start := pprof.Lookup("goroutine")

		for block := 10; block < 100; block++ {
			et.RegisterOnElection(ctx, primitives.BlockHeight(block), 0, cb)
			time.Sleep(time.Millisecond)
		}

		require.True(t, callCount > 1, "the callback must be called more than once")
		time.Sleep(100 * time.Millisecond) // a yield to let it close all goroutines
		end := pprof.Lookup("goroutine")

		if start.Count() > end.Count() {
			return
		}

		if start.Count() != end.Count() {
			t.Logf("START goroutines, count=%d", start.Count())
			start.WriteTo(&writer, 2)
			t.Logf("END goroutines, count=%d", end.Count())
			end.WriteTo(&writer, 2)
		}
		require.Equal(t, start.Count(), end.Count(), "goroutine number should be the same")
	})
}

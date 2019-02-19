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

// TODO Consider removing this test entirely - sleeps in tests are bad
func TestCallbackTrigger(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		et := buildElectionTrigger(ctx, t, 50*time.Millisecond)

		wasCalled := false
		cb := func(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, onElectionCB func(m lhmetrics.ElectionMetrics)) {
			wasCalled = true
		}
		et.RegisterOnElection(ctx, 20, 0, cb)

		time.Sleep(80 * time.Millisecond)

		require.True(t, wasCalled, "Did not call the timer callback")
	})
}

func TestCallbackTriggerOnce(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		et := buildElectionTrigger(ctx, t, 10*time.Millisecond)

		callCount := 0
		cb := func(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, onElectionCB func(m lhmetrics.ElectionMetrics)) {
			callCount++
		}
		et.RegisterOnElection(ctx, 10, 0, cb)

		time.Sleep(25 * time.Millisecond)

		require.Exactly(t, 1, callCount, "Trigger callback called more than once")
	})
}

func TestCallbackTriggerTwiceInARow(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		et := buildElectionTrigger(ctx, t, 10*time.Millisecond)

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
		et := buildElectionTrigger(ctx, t, 30*time.Millisecond)

		callCount := 0
		cb := func(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, onElectionCB func(m lhmetrics.ElectionMetrics)) {
			callCount++
		}

		et.RegisterOnElection(ctx, 10, 0, cb)
		time.Sleep(10 * time.Millisecond)
		et.RegisterOnElection(ctx, 10, 0, cb)
		time.Sleep(10 * time.Millisecond)
		et.RegisterOnElection(ctx, 10, 0, cb)
		time.Sleep(20 * time.Millisecond)
		et.RegisterOnElection(ctx, 10, 0, cb)

		require.Exactly(t, 1, callCount, "Trigger callback called more than once")
	})
}

func TestNotTriggerIfSameViewButDifferentHeight(t *testing.T) {
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
		time.Sleep(3 * time.Millisecond)

		et.RegisterOnElection(ctx, 11, 0, cb2)
		time.Sleep(50 * time.Millisecond)

		require.False(t, beforeSecondRegister, "should not trigger the first one")
		require.True(t, afterSecondRegister, "should only trigger the second one")

	})
}

func TestNotTriggerIfSameHeightButDifferentView(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		et := buildElectionTrigger(ctx, t, 30*time.Millisecond)

		callCount := 0
		cb := func(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, onElectionCB func(m lhmetrics.ElectionMetrics)) {
			callCount++
		}

		et.RegisterOnElection(ctx, 10, 0, cb)
		time.Sleep(10 * time.Millisecond)
		et.RegisterOnElection(ctx, 10, 1, cb)
		time.Sleep(10 * time.Millisecond)
		et.RegisterOnElection(ctx, 10, 2, cb)
		time.Sleep(10 * time.Millisecond)
		et.RegisterOnElection(ctx, 10, 3, cb)
		time.Sleep(10 * time.Millisecond)
		et.RegisterOnElection(ctx, 10, 4, cb)
		time.Sleep(10 * time.Millisecond)
		et.RegisterOnElection(ctx, 10, 5, cb)

		require.Exactly(t, 0, callCount, "Trigger callback called")
	})
}

func TestViewChanges(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		et := buildElectionTrigger(ctx, t, 50*time.Millisecond)

		wasCalled := false
		cb := func(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, onElectionCB func(m lhmetrics.ElectionMetrics)) {
			wasCalled = true
		}

		et.RegisterOnElection(ctx, 10, 0, cb) // 2 ** 0 * 20 = 20
		time.Sleep(10 * time.Millisecond)

		et.RegisterOnElection(ctx, 10, 1, cb) // 2 ** 1 * 20 = 40
		time.Sleep(30 * time.Millisecond)

		et.RegisterOnElection(ctx, 10, 2, cb) // 2 ** 2 * 20 = 80
		time.Sleep(70 * time.Millisecond)

		et.RegisterOnElection(ctx, 10, 3, cb) // 2 ** 3 * 20 = 160

		require.False(t, wasCalled, "Trigger the callback even if a new Register was called with a new view")
	})
}

func TestViewPowTimeout(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		et := buildElectionTrigger(ctx, t, 10*time.Millisecond)

		wasCalled := false
		cb := func(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, onElectionCB func(m lhmetrics.ElectionMetrics)) {
			wasCalled = true
		}

		et.RegisterOnElection(ctx, 10, 2, cb) // 2 ** 2 * 10 = 40
		time.Sleep(30 * time.Millisecond)
		require.False(t, wasCalled, "Triggered the callback too early")
		time.Sleep(30 * time.Millisecond)
		require.True(t, wasCalled, "Did not trigger the callback after the required timeout")
	})
}

type testWriter struct {
	t *testing.T
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	w.t.Log(string(p))
	return 0, nil
}

func NewTestWriter(t *testing.T) *testWriter {
	return &testWriter{
		t,
	}
}

func TestElectionTriggerDoesNotLeak(t *testing.T) {
	// this test checks that after multiple registrations, there are no goroutine leaks
	writer := NewTestWriter(t)
	test.WithContext(func(ctx context.Context) {
		et := buildElectionTrigger(ctx, t, time.Millisecond)

		callCount := 0
		cb := func(ctx context.Context, blockHeight primitives.BlockHeight, view primitives.View, onElectionCB func(m lhmetrics.ElectionMetrics)) {
			callCount++
		}
		start := pprof.Lookup("goroutine")

		for block := 10; block < 100; block++ {
			et.RegisterOnElection(ctx, primitives.BlockHeight(block), 0, cb)
			time.Sleep(2 * time.Millisecond)
		}

		end := pprof.Lookup("goroutine")

		require.True(t, callCount > 1, "the callback must be called more than once") // sanity

		if start.Count() > end.Count() {
			return
		}

		if start.Count() != end.Count() {
			t.Logf("START goroutines, count=%d", start.Count())
			start.WriteTo(writer, 2)
			t.Logf("END goroutines, count=%d", end.Count())
			end.WriteTo(writer, 2)
		}
		require.Equal(t, start.Count(), end.Count(), "goroutine number should be the same")
	})
}

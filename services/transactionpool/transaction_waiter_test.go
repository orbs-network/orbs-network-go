package transactionpool

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestTransactionWaiterReturnsWhenContextIsDone(t *testing.T) {
	test.WithContext(func(parent context.Context) {
		timeout, cancel := context.WithTimeout(parent, 1*time.Second)
		defer cancel()

		cancelledContext, cancelInner := context.WithCancel(parent)
		cancelInner()

		waiter := newTransactionWaiter()

		ch := waiter.waitFor(cancelledContext, 1)

		select {
		case endCond := <-ch:
			require.False(t, endCond, "waiter returned for wrong reason")
		case <-timeout.Done():
			t.Fatalf("timed out before waiter returned")
		}
	})
}

func TestTransactionWaiterDoesNotReturnIfIncrementedLessThanExpected(t *testing.T) {
	test.WithContext(func(parent context.Context) {
		timeout, cancel := context.WithTimeout(parent, 1*time.Second)
		defer cancel()

		cancellableContext, cancelWaitingContext := context.WithCancel(parent)

		waiter := newTransactionWaiter()

		ch := waiter.waitFor(cancellableContext, 2)
		waiter.inc(parent)
		cancelWaitingContext()

		select {
		case endCond := <-ch:
			require.False(t, endCond, "waiter returned for wrong reason")
		case <-timeout.Done():
			t.Fatalf("timed out before waiter returned")
		}
	})
}

func TestTransactionWaiterReturnsAfterThresholdIsMet(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		timeout, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()

		waiter := newTransactionWaiter()

		ch := waiter.waitFor(ctx, 2)

		waiter.inc(ctx)
		waiter.inc(ctx)

		select {
		case endCond := <-ch:
			require.True(t, endCond, "waiter returned for wrong reason")
		case <-timeout.Done():
			t.Fatalf("timed out before waiter returned")
		}
	})
}

func TestTransactionWaiterDoesNotBlockIncWhenNoOneIsWaiting(t *testing.T) {
	test.WithContext(func(parent context.Context) {
		timeout, cancel := context.WithTimeout(parent, 1*time.Second)
		defer cancel()

		waiter := newTransactionWaiter()
		ch := make(chan struct{})

		go func() {
			waiter.inc(parent)
			close(ch)
		}()

		select {
		case <-ch:
			// did not block, end test successfully
		case <-timeout.Done():
			t.Fatalf("increment blocked when no one was waiting")
		}
	})
}

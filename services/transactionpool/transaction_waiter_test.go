// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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
		cancelledContext, cancelInner := context.WithCancel(parent)
		cancelInner()

		waiter := newTransactionWaiter()

		require.False(t, waiter.waitForIncomingTransaction(cancelledContext), "waiter returned for wrong reason")
	})
}

func TestTransactionWaiterReturnsTrueWhenIncomingTransaction(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		timeout, cancel := context.WithTimeout(ctx, 100*time.Second)
		defer cancel()

		waiter := newTransactionWaiter()

		ch := make(chan bool)
		go func() {
			ch <- waiter.waitForIncomingTransaction(ctx)
		}()

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

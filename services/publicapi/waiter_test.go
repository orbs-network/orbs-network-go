// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package publicapi

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestPublicApiWaiter_Add(t *testing.T) {
	t.Parallel()
	test.WithContext(func(ctx context.Context) {
		waiter := newWaiter()
		wc := waiter.add("key")

		require.NotNil(t, wc, "wait object is nil when it should exist")
	})
}

func TestPublicApiWaiter_AddTwice(t *testing.T) {
	t.Parallel()
	test.WithContext(func(ctx context.Context) {
		waiter := newWaiter()
		wc1 := waiter.add("key")
		wc2 := waiter.add("key")

		require.NotNil(t, wc1, "wait object is nil when it should exist")
		require.NotNil(t, wc2, "wait object is nil when it should exist")
		require.NotEqual(t, wc1, wc2, "both wait objects must be different")
		require.Equal(t, 1, len(waiter.m), "must have one key-value pair in upper level")
	})
}

func TestPublicApiWaiter_AddTwoKeys(t *testing.T) {
	t.Parallel()
	test.WithContext(func(ctx context.Context) {
		waiter := newWaiter()
		waiter.add("key1")
		waiter.add("key2")

		require.Equal(t, 2, len(waiter.m), "must have two key-value pair in upper level")
	})
}

func TestPublicApiWaiter_DeleteKey(t *testing.T) {
	t.Parallel()
	test.WithContext(func(ctx context.Context) {
		waiter := newWaiter()
		wc1 := waiter.add("key1")
		waiter.add("key2")
		wcs1 := waiter._deleteByKey("key1")

		require.Equal(t, 1, len(waiter.m), "must have one key-value pair in upper level")
		require.NotNil(t, wcs1, "wait object channels is nil when it should exist")
		_, exists := wcs1[wc1]
		require.True(t, exists, "the deleted channel was destroyed when it was suppose to be returned")
	})
}

func TestPublicApiWaiter_DeleteChan(t *testing.T) {
	t.Parallel()
	test.WithContext(func(ctx context.Context) {
		key := "key"
		waiter := newWaiter()
		wc1 := waiter.add(key)
		wc2 := waiter.add(key)
		waiter.deleteByChannel(wc1)

		require.Equal(t, 1, len(waiter.m), "must have one key-value pair in upper level")
		require.Equal(t, 1, len(waiter.m[key]), "must have one channel left in lower level")

		_, exists := waiter.m[key][wc2]
		require.True(t, exists, "second chan must still exits")
	})
}

func TestPublicApiWaiter_DeleteAllChan(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		waiter := newWaiter()
		wc1 := waiter.add("key")
		wc2 := waiter.add("key")
		waiter.deleteByChannel(wc1)
		waiter.deleteByChannel(wc2)

		require.Equal(t, 0, len(waiter.m), "must be empty")
	})
}

func TestPublicApiWaiter_WaitFor(t *testing.T) {
	t.Parallel()
	test.WithContext(func(ctx context.Context) {
		waiter := newWaiter()
		wc := waiter.add("key")

		ctx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
		defer cancel()

		_, err := waiter.wait(ctx, wc)
		require.Error(t, err, "expected waiting to be aborted")
		require.Contains(t, err.Error(), "waiting aborted due to context termination for key", "expected waiting to be aborted with timeout")
	})
}

func TestPublicApiWaiter_CompleteAllChannels(t *testing.T) {
	t.Parallel()
	test.WithContext(func(ctx context.Context) {
		key := "key"
		waiter := newWaiter()
		wc1 := waiter.add(key)
		wc2 := waiter.add(key)

		done := make(chan struct{}, 2)

		go waitHarness(ctx, t, "1", waiter, wc1, 100*time.Millisecond, false, done)
		go waitHarness(ctx, t, "2", waiter, wc2, 100*time.Millisecond, false, done)

		waiter.complete(key, "hello")
		<-done
		<-done
		_, open := <-wc1.c
		require.False(t, open, "channel 1 should be closed")
		_, open = <-wc2.c
		require.False(t, open, "channel 2 should be closed")
	})
}

func TestPublicApiWaiter_CompleteChanWhenOtherIsDeletedDuringWait(t *testing.T) {
	t.Parallel()
	test.WithContext(func(ctx context.Context) {
		key := "key"
		waiter := newWaiter()
		wc1 := waiter.add(key)
		wc2 := waiter.add(key)

		done := make(chan struct{}, 2)

		go waitHarness(ctx, t, "1", waiter, wc1, 10*time.Millisecond, true, done)
		go waitHarness(ctx, t, "2", waiter, wc2, 10*time.Millisecond, false, done)

		waiter.deleteByChannel(wc1) // as if it was returned error quickly
		waiter.complete(key, "hello")
		<-done
		<-done

		_, open := <-wc1.c
		require.False(t, open, "channel 1 should be closed")
		_, open = <-wc2.c
		require.False(t, open, "channel 2 should be closed")
	})
}

func TestPublicApiWaiter_CompleteOnBothWhenOneIsCanceled(t *testing.T) {
	t.Parallel()
	test.WithContext(func(ctx context.Context) {
		key := "key"
		waiter := newWaiter()
		wc1 := waiter.add(key)
		wc2 := waiter.add(key)

		done := make(chan struct{}, 2)

		ctx2, cancel := context.WithCancel(ctx)
		cancel()

		go waitHarness(ctx2, t, "1", waiter, wc1, 1*time.Second, true, done)
		go waitHarness(ctx, t, "2", waiter, wc2, 1*time.Second, false, done)

		<-done // force waiting till first go routine finishes in the "cancel" capacity.
		waiter.complete(key, "hello")
		<-done

		_, open := <-wc1.c
		require.False(t, open, "channel 1 should be closed")
		_, open = <-wc2.c
		require.False(t, open, "channel 2 should be closed")
	})
}

func TestPublicApiWaiter_WaitGracefulShutdown(t *testing.T) {
	t.Parallel()
	var waiter *waiter
	done := make(chan struct{})

	test.WithContext(func(ctx context.Context) {
		key := "key"
		waiter = newWaiter()
		wc1 := waiter.add(key)
		wc2 := waiter.add(key)

		var waitTillCancelled = func(wc *waiterChannel) {
			ctxWithTimeout, cancel := context.WithTimeout(ctx, 1*time.Second)
			defer cancel()

			startTime := time.Now()
			wo, err := waiter.wait(ctxWithTimeout, wc)
			assert.Error(t, err, "expected waiting to be aborted")
			assert.WithinDuration(t, time.Now(), startTime, 100*time.Millisecond, "expected not to reach timeout")
			require.Nil(t, wo, "wait object (1) should be nil")
			done <- struct{}{}
		}

		go waitTillCancelled(wc1)
		go waitTillCancelled(wc2)
	})
	<-done
	<-done
}

func waitHarness(ctx context.Context, t *testing.T, name string, waiter *waiter, waitResult *waiterChannel, duration time.Duration, shouldErr bool, done chan struct{}) {
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	wo, err := waiter.wait(ctx, waitResult)
	if shouldErr {
		assert.Error(t, err, "expected error to happen for (%s)", name)
		require.Nil(t, wo, "wait object (%s) is not nil when it should", name)
	} else {
		assert.NoError(t, err, "expected error to not occur for (%s)", name)
		require.NotNil(t, wo, "wait object (%s) is nil when it should", name)
	}
	done <- struct{}{}
}

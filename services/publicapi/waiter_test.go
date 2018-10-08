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
		waiter := newWaiter(ctx)
		wc := waiter.add("key")

		require.NotNil(t, wc, "wait object is nil when it should exist")
	})
}

func TestPublicApiWaiter_AddTwice(t *testing.T) {
	t.Parallel()
	test.WithContext(func(ctx context.Context) {
		waiter := newWaiter(ctx)
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
		waiter := newWaiter(ctx)
		waiter.add("key1")
		waiter.add("key2")

		require.Equal(t, 2, len(waiter.m), "must have two key-value pair in upper level")
	})
}

func TestPublicApiWaiter_DeleteKey(t *testing.T) {
	t.Parallel()
	test.WithContext(func(ctx context.Context) {
		waiter := newWaiter(ctx)
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
		waiter := newWaiter(ctx)
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
		waiter := newWaiter(ctx)
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
		waiter := newWaiter(ctx)
		wc := waiter.add("key")

		_, err := waiter.wait(wc, 10*time.Millisecond)
		require.Error(t, err, "expected waiting to be aborted")
		require.Contains(t, err.Error(), "timed out waiting for result", "expected waiting to be aborted with timeout")
	})
}

func TestPublicApiWaiter_CompleteAllChannels(t *testing.T) {
	t.Parallel()
	test.WithContext(func(ctx context.Context) {
		key := "key"
		waiter := newWaiter(ctx)
		wc1 := waiter.add(key)
		wc2 := waiter.add(key)

		c := make(chan struct{}, 2)

		go func() {
			wo, err := waiter.wait(wc1, 100*time.Millisecond)
			assert.NoError(t, err)
			require.NotNil(t, wo, "wait object (1) is nil when it should")
			c <- struct{}{}
		}()

		go func() {
			wo, err := waiter.wait(wc2, 100*time.Millisecond)
			assert.NoError(t, err)
			require.NotNil(t, wo, "wait object (2) is nil when it should")
			c <- struct{}{}
		}()

		waiter.complete(key, &waiterObject{"hello"})
		<-c
		<-c
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
		waiter := newWaiter(ctx)
		wc1 := waiter.add(key)
		wc2 := waiter.add(key)

		c := make(chan struct{}, 2)
		go func() {
			wo, err := waiter.wait(wc1, 100*time.Millisecond)
			assert.Error(t, err)
			require.Nil(t, wo, "wait object (1) should be nil")
			c <- struct{}{}
		}()

		go func() {
			wo, err := waiter.wait(wc2, 100*time.Millisecond)
			assert.NoError(t, err)
			require.NotNil(t, wo, "wait object (2) is not nil when it should")
			c <- struct{}{}
		}()

		waiter.deleteByChannel(wc1) // as if it was returned error quickly
		waiter.complete(key, &waiterObject{"hello"})
		<-c
		<-c

		_, open := <-wc1.c
		require.False(t, open, "channel 1 should be closed")
		_, open = <-wc2.c
		require.False(t, open, "channel 2 should be closed")
	})
}

func TestPublicApiWaiter_CompleteChanWhenOtherIsTimedOut(t *testing.T) {
	t.Parallel()
	test.WithContext(func(ctx context.Context) {
		key := "key"
		waiter := newWaiter(ctx)
		wc1 := waiter.add(key)
		wc2 := waiter.add(key)

		c := make(chan struct{}, 2)
		go func() {
			wo, err := waiter.wait(wc1, 5*time.Millisecond)
			assert.Error(t, err)
			require.Nil(t, wo, "wait object (1) should be nil")
			c <- struct{}{}
		}()

		go func() {
			wo, err := waiter.wait(wc2, 1*time.Second)
			assert.NoError(t, err)
			require.NotNil(t, wo, "wait object (2) is not nil when it should")
			c <- struct{}{}
		}()

		time.Sleep(15 * time.Millisecond)
		waiter.complete(key, &waiterObject{"hello"})
		<-c
		<-c

		_, open := <-wc1.c
		require.False(t, open, "channel 1 should be closed")
		_, open = <-wc2.c
		require.False(t, open, "channel 2 should be closed")
	})
}

func TestPublicApiWaiter_WaitGracefulShutdown(t *testing.T) {
	t.Parallel()
	var waiter *waiter
	c := make(chan struct{})

	test.WithContext(func(ctx context.Context) {
		key := "key"
		waiter = newWaiter(ctx)
		wc1 := waiter.add(key)
		wc2 := waiter.add(key)

		var waitTillCancelled = func(wc *waiterChannel) {
			startTime := time.Now()
			wo, err := waiter.wait(wc, 1*time.Second)
			assert.Error(t, err, "expected waiting to be aborted")
			assert.WithinDuration(t, time.Now(), startTime, 100*time.Millisecond, "expected not to reach timeout")
			require.Nil(t, wo, "wait object (1) should be nil")
			c <- struct{}{}
		}

		go waitTillCancelled(wc1)
		go waitTillCancelled(wc2)
	})
	<-c
	<-c
}

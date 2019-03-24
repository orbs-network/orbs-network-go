// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package transactionpool

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

var tickInterval = func() time.Duration { return 1 * time.Millisecond }
var expiration = func() time.Duration { return 30 * time.Minute }

func TestStopsWhenContextIsCancelled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())

	m := aCleaner()
	stopped := startCleaningProcess(ctx, tickInterval, expiration, m, func() (primitives.BlockHeight, primitives.TimestampNano) { return 0, 0 }, nil)

	cancel()

	<-stopped
}

func TestTicksOnSchedule(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())

	ts := primitives.TimestampNano(time.Now().UnixNano())

	m := aCleaner()
	stopped := startCleaningProcess(ctx, tickInterval, expiration, m, func() (primitives.BlockHeight, primitives.TimestampNano) { return 0, ts }, nil)

	// waiting multiple times to assert that ticker is looping :)
	for i := 0; i < 3; i++ {
		select {
		case cleaned := <-m.cleaned:
			require.InDelta(t, int64(ts-primitives.TimestampNano(expiration())), int64(cleaned), float64(1*time.Second), "did not call cleaner with expected time")
		case <-time.After(tickInterval() * 100):
			t.Fatalf("did not call cleaner within expected timeframe")
		}

	}

	cancel()

	<-stopped
}

type mockCleaner struct {
	cleaned chan primitives.TimestampNano
}

func (c *mockCleaner) clearTransactionsOlderThan(ctx context.Context, ts primitives.TimestampNano) {
	c.cleaned <- ts
}

func aCleaner() *mockCleaner {
	return &mockCleaner{cleaned: make(chan primitives.TimestampNano, 1)}
}

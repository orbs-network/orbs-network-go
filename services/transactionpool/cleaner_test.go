package transactionpool

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

var tickInterval = func() time.Duration { return 100 * time.Microsecond }
var expiration = func() time.Duration { return 30 * time.Minute }

func TestStopsWhenContextIsCancelled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())

	m := aCleaner()
	stopped := startCleaningProcess(ctx, tickInterval, expiration, m, nil)

	cancel()

	<-stopped
}

func TestTicksOnSchedule(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())

	m := aCleaner()
	stopped := startCleaningProcess(ctx, tickInterval, expiration, m, nil)

	// waiting multiple times to assert that ticker is looping :)
	for i := 0; i < 3; i++ {
		select {
		case cleaned := <-m.cleaned:
			require.InDelta(t, time.Now().Add(-1*expiration()).UnixNano(), cleaned.UnixNano(), float64(1*time.Second), "did not call cleaner with expected time")
		case <-time.After(tickInterval() * 100):
			t.Fatalf("did not call cleaner within expected timeframe")
		}

	}

	cancel()

	<-stopped
}

type mockCleaner struct {
	cleaned chan time.Time
}

func (c *mockCleaner) clearTransactionsOlderThan(time time.Time) {
	c.cleaned <- time
}

func aCleaner() *mockCleaner {
	return &mockCleaner{cleaned: make(chan time.Time, 1)}
}

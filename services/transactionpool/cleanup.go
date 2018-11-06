package transactionpool

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"time"
)

type cleaner interface {
	clearTransactionsOlderThan(ctx context.Context, time time.Time)
}

func startCleaningProcess(ctx context.Context, tickInterval func() time.Duration, expiration func() time.Duration, c cleaner, logger log.BasicLogger) chan struct{} {
	stopped := make(chan struct{})
	synchronization.NewPeriodicalTrigger(ctx, tickInterval(), logger, func() {
		c.clearTransactionsOlderThan(ctx, time.Now().Add(-1*expiration()))
	}, func() {
		close(stopped)
	})

	return stopped
}

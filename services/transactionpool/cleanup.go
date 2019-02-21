package transactionpool

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"time"
)

type cleaner interface {
	clearTransactionsOlderThan(ctx context.Context, timestamp primitives.TimestampNano)
}

func startCleaningProcess(ctx context.Context, tickInterval func() time.Duration, expiration func() time.Duration, c cleaner, lastBlockHeightAndTime func() (primitives.BlockHeight, primitives.TimestampNano), logger log.BasicLogger) chan struct{} {
	stopped := make(chan struct{})
	synchronization.NewPeriodicalTrigger(ctx, tickInterval(), logger, func() {
		_, lastCommittedBlockTime := lastBlockHeightAndTime()
		c.clearTransactionsOlderThan(ctx, lastCommittedBlockTime-primitives.TimestampNano(expiration().Nanoseconds()))
	}, func() {
		close(stopped)
	})

	return stopped
}

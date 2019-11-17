// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package transactionpool

import (
	"context"
	"fmt"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
	"time"
)

type cleaner interface {
	clearTransactionsOlderThan(ctx context.Context, timestamp primitives.TimestampNano)
}

func startCleaningProcess(ctx context.Context, poolName string, tickInterval func() time.Duration, expiration func() time.Duration, c cleaner, lastBlockHeightAndTime func() (primitives.BlockHeight, primitives.TimestampNano), logger log.Logger) govnr.ShutdownWaiter {
	return synchronization.NewPeriodicalTrigger(ctx, fmt.Sprintf("%s cleaner trigger", poolName), synchronization.NewTimeTicker(tickInterval()), logger, func() {
		_, lastCommittedBlockTime := lastBlockHeightAndTime()
		c.clearTransactionsOlderThan(ctx, lastCommittedBlockTime-primitives.TimestampNano(expiration().Nanoseconds()))
	}, nil)
}

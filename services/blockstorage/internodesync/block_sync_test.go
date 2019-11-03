// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package internodesync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test/with"
	"testing"
)

func TestBlockSyncStartsWithImmediateSync(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		h := newBlockSyncHarnessWithManualNoCommitTimeoutTimer(parent.Logger, func() *synchronization.Timer {
			return synchronization.NewTimerWithManualTick()
		})
		h.expectSyncOnStart()
		parent.Supervise(newBlockSyncWithFactory(ctx, h.factory, h.gossip, h.storage, parent.Logger, h.metricFactory))
		h.eventuallyVerifyMocks(t, 2) // just need to verify we used gossip/storage for sync
	})
}

func TestBlockSyncStaysInIdleOnBlockCommitExternalMessage(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		manualIdleStateTimeoutTimers := make(chan *synchronization.Timer)
		h := newBlockSyncHarnessWithManualNoCommitTimeoutTimer(parent.Logger, func() *synchronization.Timer {
			currentTimer := synchronization.NewTimerWithManualTick()
			manualIdleStateTimeoutTimers <- currentTimer
			return currentTimer
		})

		h.expectSyncOnStart()

		bs := newBlockSyncWithFactory(ctx, h.factory, h.gossip, h.storage, parent.Logger, h.metricFactory)
		parent.Supervise(bs)

		firstIdleStateTimeoutTimer := <-manualIdleStateTimeoutTimers // reach first idle state
		h.eventuallyVerifyMocks(t, 2)                                // short eventually                                            // confirm init sync attempt occurred (expected mock calls)

		bs.HandleBlockCommitted(ctx) // trigger transition (from idle state) to a new idle state

		<-manualIdleStateTimeoutTimers // reach second idle state

		firstIdleStateTimeoutTimer.ManualTick() // simulate no-commit-timeout for the first idle state object
		h.consistentlyVerifyMocks(t, 4, "expected no new sync attempts to occur after a timeout expires on a stale idle state")

		select {
		case <-manualIdleStateTimeoutTimers:
			t.Fatal("expected state machine to NOT renew idle timer without commits or no-commit-timeouts triggered")
		default:
		}

	})
}

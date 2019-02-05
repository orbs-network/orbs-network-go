package internodesync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBlockSyncStartsWithImmediateSync(t *testing.T) {
	h := newBlockSyncHarnessWithManualNoCommitTimeoutTimer(t, func() *synchronization.Timer {
		return synchronization.NewTimerWithManualTick()
	})

	var bs *BlockSync
	test.WithContext(func(ctx context.Context) {
		h.expectSyncOnStart()

		bs = newBlockSyncWithFactory(ctx, h.factory, h.gossip, h.storage, h.logger, h.metricFactory)

		h.eventuallyVerifyMocks(t, 2) // just need to verify we used gossip/storage for sync
	})

	shutdown := h.waitForShutdown(bs)
	require.True(t, shutdown, "expecting state to be set to nil (=shutdown)")
}

func TestBlockSyncStaysInIdleOnBlockCommitExternalMessage(t *testing.T) {
	manualIdleTimeoutTimerChan := make(chan *synchronization.Timer)
	h := newBlockSyncHarnessWithManualNoCommitTimeoutTimer(t, func() *synchronization.Timer {
		currentTimer := synchronization.NewTimerWithManualTick()
		manualIdleTimeoutTimerChan <- currentTimer
		return currentTimer
	})

	var bs *BlockSync
	test.WithContext(func(ctx context.Context) {
		h.expectSyncOnStart()

		bs = newBlockSyncWithFactory(ctx, h.factory, h.gossip, h.storage, h.logger, h.metricFactory)

		firstIdleStateTimeoutTimer := <-manualIdleTimeoutTimerChan // reach first idle state
		h.verifyMocks(t)                                           // confirm init sync attempt occurred (expected mock calls)

		bs.HandleBlockCommitted(ctx) // trigger transition (from idle state) to a new idle state

		<-manualIdleTimeoutTimerChan // reach second idle state

		firstIdleStateTimeoutTimer.ManualTick() // simulate no-commit-timeout for the first idle state object
		h.consistentlyVerifyMocks(t, 4, "expected no new sync attempts to occur after a timeout expires on a stale idle state")
	})

	shutdown := h.waitForShutdown(bs)
	require.True(t, shutdown, "expecting state to be set to nil (=shutdown)")
}

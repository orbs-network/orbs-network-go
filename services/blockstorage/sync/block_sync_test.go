package sync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestBlockSyncStartsWithImmediateSync(t *testing.T) {
	h := newBlockSyncHarnessWithManualNoCommitTimeoutTimer(func() *synchronization.Timer {
		return synchronization.NewTimerWithManualTick()
	})

	var bs *BlockSync
	test.WithContext(func(ctx context.Context) {
		h.expectSyncOnStart()

		cfg := newBlockSyncConfigForTestsWithInfiniteTimeouts() // don't want timeouts since manual timer
		bs = newBlockSyncWithFactory(ctx, h.factory, cfg, h.gossip, h.storage, h.logger, h.metricFactory)

		h.eventuallyVerifyMocks(t, 2) // just need to verify we used gossip/storage for sync
	})

	shutdown := h.waitForShutdown(bs)
	require.True(t, shutdown, "expecting state to be set to nil (=shutdown)")
}

func TestBlockSyncStaysInIdleOnBlockCommitExternalMessage(t *testing.T) {
	manualNoCommitTimers := []*synchronization.Timer{}
	h := newBlockSyncHarnessWithManualNoCommitTimeoutTimer(func() *synchronization.Timer {
		timer := synchronization.NewTimerWithManualTick()
		manualNoCommitTimers = append(manualNoCommitTimers, timer)
		return timer
	})

	var bs *BlockSync
	test.WithContext(func(ctx context.Context) {
		h.expectSyncOnStart()

		cfg := newBlockSyncConfigForTestsWithInfiniteTimeouts() // don't want timeouts since manual timer
		bs = newBlockSyncWithFactory(ctx, h.factory, cfg, h.gossip, h.storage, h.logger, h.metricFactory)

		ok := test.Eventually(50*time.Millisecond, func() bool {
			if len(manualNoCommitTimers) > 0 {
				bs.HandleBlockCommitted(ctx)         // exit the first idle state by committing a block
				manualNoCommitTimers[0].ManualTick() // manual tick of no commit timer should do nothing for the first idle state now
				return true
			}
			return false
		})
		require.True(t, ok, "no commit timer of the first idle state should be created")

		h.consistentlyVerifyMocks(t, 2) // just need to verify we used gossip/storage for sync
	})

	shutdown := h.waitForShutdown(bs)
	require.True(t, shutdown, "expecting state to be set to nil (=shutdown)")
}

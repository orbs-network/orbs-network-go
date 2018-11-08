package sync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIdleStateStaysIdleOnCommit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		manualNoCommitTimer := synchronization.NewTimerWithManualTick()
		h := newBlockSyncHarnessWithManualNoCommitTimeoutTimer(manualNoCommitTimer)

		idle := h.factory.CreateIdleState()
		nextState := h.processStateAndWaitUntilFinished(ctx, idle, func() {
			// letting the goroutine start above
			idle.blockCommitted(ctx)
			manualNoCommitTimer.ManualTick()
		})

		require.IsType(t, &idleState{}, nextState, "nextState should still be idle")
		require.True(t, nextState != idle, "processState state should be a different idle state (which was recreated so the timer starts from be beginning)")
	})
}

func TestIdleStateMovesToCollectingOnNoCommitTimeout(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness()
		idle := h.factory.CreateIdleState()
		next := idle.processState(ctx)
		require.IsType(t, &collectingAvailabilityResponsesState{}, next, "processState state should be collecting availability responses")
	})
}

func TestIdleStateTerminatesOnContextTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	h := newBlockSyncHarness()
	cancel()
	idle := h.factory.CreateIdleState()
	next := idle.processState(ctx)

	require.Nil(t, next, "context termination should return a nil new state")
}

func TestIdleStateDoesNotBlockOnNewBlockNotificationWhenChannelIsNotReady(t *testing.T) {
	h := newBlockSyncHarness()
	test.WithContextWithTimeout(h.config.noCommit/2, func(ctx context.Context) {
		idle := h.factory.CreateIdleState()
		idle.blockCommitted(ctx) // we did not call process, so channel is not ready, test only fails on timeout, if this blocks
	})
}

func TestIdleNOP(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness()
		idle := h.factory.CreateIdleState()
		// these calls should do nothing, this is just a sanity that they do not panic and return nothing
		idle.gotAvailabilityResponse(ctx, nil)
		idle.gotBlocks(ctx, nil)
	})
}

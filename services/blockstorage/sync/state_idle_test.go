package sync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStateIdle_StaysIdleOnCommit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		manualNoCommitTimer := synchronization.NewTimerWithManualTick()
		h := newBlockSyncHarnessWithManualNoCommitTimeoutTimer(func() *synchronization.Timer {
			return manualNoCommitTimer
		})

		idle := h.factory.CreateIdleState()
		nextState := h.processStateInBackgroundAndWaitUntilFinished(ctx, idle, func() {
			idle.blockCommitted(ctx)
			manualNoCommitTimer.ManualTick() // not required, added for completion (like in state_availability_requests_test)
		})

		require.IsType(t, &idleState{}, nextState, "nextState should still be idle")
		require.True(t, nextState != idle, "processState state should be a different idle state (which was recreated so the timer starts from be beginning)")
	})
}

func TestStateIdle_MovesToCollectingAvailabilityResponsesOnNoCommitTimeout(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness()
		idle := h.factory.CreateIdleState()
		next := idle.processState(ctx)
		require.IsType(t, &collectingAvailabilityResponsesState{}, next, "processState state should be collecting availability responses")
	})
}

func TestStateIdle_TerminatesOnContextTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	h := newBlockSyncHarness()
	cancel()
	idle := h.factory.CreateIdleState()
	next := idle.processState(ctx)

	require.Nil(t, next, "context termination should return a nil new state")
}

func TestStateIdle_DoesNotBlockOnNewBlockNotificationWhenChannelIsNotReady(t *testing.T) {
	h := newBlockSyncHarness()
	test.WithContextWithTimeout(h.config.noCommit/2, func(ctx context.Context) {
		idle := h.factory.CreateIdleState()
		idle.blockCommitted(ctx) // we did not call process, so channel is not ready, test only fails on timeout, if this blocks
	})
}

func TestStateIdle_NOP(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness()
		idle := h.factory.CreateIdleState()
		// these calls should do nothing, this is just a sanity that they do not panic and return nothing
		idle.gotAvailabilityResponse(ctx, nil)
		idle.gotBlocks(ctx, nil)
	})
}

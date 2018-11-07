package sync

import (
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIdleStateStaysIdleOnCommit(t *testing.T) {
	manualNoCommitTimer := synchronization.NewTimerWithManualTick()
	h := newBlockSyncHarnessWithManualNoCommitTimeoutTimer(manualNoCommitTimer)

	idle := h.factory.CreateIdleState()
	nextState := h.processStateAndWaitUntilFinished(idle, func() {
		// letting the goroutine start above
		idle.blockCommitted(h.ctx)
		manualNoCommitTimer.ManualTick()
	})

	require.IsType(t, &idleState{}, nextState, "nextState should still be idle")
	require.True(t, nextState != idle, "processState state should be a different idle state (which was recreated so the timer starts from be beginning)")
}

func TestIdleStateMovesToCollectingOnNoCommitTimeout(t *testing.T) {
	h := newBlockSyncHarness()
	idle := h.factory.CreateIdleState()
	next := idle.processState(h.ctx)
	require.IsType(t, &collectingAvailabilityResponsesState{}, next, "processState state should be collecting availability responses")
}

func TestIdleStateTerminatesOnContextTermination(t *testing.T) {
	h := newBlockSyncHarness()
	h.cancel()
	idle := h.factory.CreateIdleState()
	next := idle.processState(h.ctx)

	require.Nil(t, next, "context termination should return a nil new state")
}

func TestIdleStateDoesNotBlockOnNewBlockNotificationWhenChannelIsNotReady(t *testing.T) {
	h := newBlockSyncHarness()
	h = h.withCtxTimeout(h.config.noCommit / 2)
	idle := h.factory.CreateIdleState()
	idle.blockCommitted(h.ctx) // we did not call process, so channel is not ready, test only fails on timeout, if this blocks
	h.cancel()
}

func TestIdleNOP(t *testing.T) {
	h := newBlockSyncHarness()
	idle := h.factory.CreateIdleState()
	// these calls should do nothing, this is just a sanity that they do not panic and return nothing
	idle.gotAvailabilityResponse(h.ctx, nil)
	idle.gotBlocks(h.ctx, nil)
}

package sync

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestIdleStateStaysIdleOnCommit(t *testing.T) {
	h := newBlockSyncHarness().withNoCommitTimeout(time.Second) // we are checking for a newly created state, the timeout here is irrelevant
	idle := h.sf.CreateIdleState()
	next := h.nextState(idle, func() {
		// letting the goroutine start above
		time.Sleep(time.Millisecond)
		idle.blockCommitted()
	})

	require.IsType(t, &idleState{}, next, "next should still be idle")
	require.True(t, next != idle, "processState state should be a different idle state (which was recreated so the timer starts from be beginning)")
}

func TestIdleStateMovesToCollectingOnNoCommitTimeout(t *testing.T) {
	h := newBlockSyncHarness()
	idle := h.sf.CreateIdleState()
	next := idle.processState(h.ctx)
	require.IsType(t, &collectingAvailabilityResponsesState{}, next, "processState state should be collecting availability responses")
}

func TestIdleStateTerminatesOnContextTermination(t *testing.T) {
	h := newBlockSyncHarness()
	h.cancel()
	idle := h.sf.CreateIdleState()
	next := idle.processState(h.ctx)

	require.Nil(t, next, "context termination should return a nil new state")
}

func TestIdleNOP(t *testing.T) {
	h := newBlockSyncHarness()
	idle := h.sf.CreateIdleState()
	// these calls should do nothing, this is just a sanity that they do not panic and return nothing
	idle.gotAvailabilityResponse(nil)
	idle.gotBlocks(nil)
}

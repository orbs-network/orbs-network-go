package sync

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIdleStateStaysIdleOnCommit(t *testing.T) {
	h := newBlockSyncHarness()
	idle := h.sf.CreateIdleState()
	var next syncState = nil
	latch := make(chan struct{})
	go func() {
		next = idle.processState(h.ctx)
		latch <- struct{}{}
	}()
	idle.blockCommitted()
	<-latch
	require.True(t, next != idle, "processState state should be a different idle state (which was restarted)")
}

func TestIdleStateMovesToCollectingOnNoCommitTimeout(t *testing.T) {
	h := newBlockSyncHarness()
	idle := h.sf.CreateIdleState()
	next := idle.processState(h.ctx)
	_, ok := next.(*collectingAvailabilityResponsesState)
	require.True(t, ok, "processState state should be collecting availability responses")
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

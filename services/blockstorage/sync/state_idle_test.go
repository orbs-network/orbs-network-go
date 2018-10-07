package sync

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestIdleStateStaysIdleOnCommit(t *testing.T) {
	h := newBlockSyncHarness().WithNoCommitTimeout(3 * time.Millisecond)
	idle := h.sf.CreateIdleState()
	var next syncState = nil
	latch := make(chan struct{})
	go func() {
		next = idle.processState(h.ctx)
		latch <- struct{}{}
	}()
	idle.blockCommitted(primitives.BlockHeight(11))
	<-latch
	require.True(t, next != idle, "processState state should be a different idle state (which was restarted)")
}

func TestIdleStateMovesToCollectingOnNoCommitTimeout(t *testing.T) {
	h := newBlockSyncHarness().WithNoCommitTimeout(3 * time.Millisecond)
	idle := h.sf.CreateIdleState()
	next := idle.processState(h.ctx)
	_, ok := next.(*collectingAvailabilityResponsesState)
	require.True(t, ok, "processState state should be collecting availability responses")
}

func TestIdleStateTerminatesOnContextTermination(t *testing.T) {
	h := newBlockSyncHarness().WithNoCommitTimeout(3 * time.Millisecond)
	h.Cancel()
	idle := h.sf.CreateIdleState()
	next := idle.processState(h.ctx)

	require.Nil(t, next, "context termination should return a nil new state")
}

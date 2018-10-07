package sync

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFinishedWithNoResponsesGoBackToIdle(t *testing.T) {
	h := newBlockSyncHarness()
	finishedState := h.sf.CreateFinishedCARState([]*gossipmessages.BlockAvailabilityResponseMessage{})
	shouldBeIdleState := finishedState.processState(h.ctx)

	_, isIdle := shouldBeIdleState.(*idleState)
	require.True(t, isIdle, "next state should be idle")
}

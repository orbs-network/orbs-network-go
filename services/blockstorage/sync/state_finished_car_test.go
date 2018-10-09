package sync

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFinishedWithNoResponsesGoBackToIdle(t *testing.T) {
	h := newBlockSyncHarness()
	finishedState := h.sf.CreateFinishedCARState([]*gossipmessages.BlockAvailabilityResponseMessage{})
	shouldBeIdleState := finishedState.processState(h.ctx)

	require.IsType(t, &idleState{}, shouldBeIdleState, "next state should be idle")
}

func TestFinishedWithResponsesMoveToWaitingForChunk(t *testing.T) {
	response := builders.BlockAvailabilityResponseInput().Build().Message
	h := newBlockSyncHarness()
	finishedState := h.sf.CreateFinishedCARState([]*gossipmessages.BlockAvailabilityResponseMessage{response})
	shouldBeWaitingState := finishedState.processState(h.ctx)

	require.IsType(t, &waitingForChunksState{}, shouldBeWaitingState, "next state should be waiting for chunk")
}

func TestFinishedWithInvalidResponsesMovesToIdle(t *testing.T) {
	h := newBlockSyncHarness()
	finishedState := h.sf.CreateFinishedCARState(nil)
	shouldBeIdleState := finishedState.processState(h.ctx)

	require.IsType(t, &idleState{}, shouldBeIdleState, "next state should be idle when invalid input")
}

func TestFinishedNOP(t *testing.T) {
	h := newBlockSyncHarness()
	finishedState := h.sf.CreateFinishedCARState([]*gossipmessages.BlockAvailabilityResponseMessage{})

	// sanity test, these should do nothing
	finishedState.gotBlocks(nil)
	finishedState.blockCommitted(primitives.BlockHeight(0))
	finishedState.gotAvailabilityResponse(nil)
}

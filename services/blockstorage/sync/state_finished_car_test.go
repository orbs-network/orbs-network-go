package sync

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFinishedWithNoResponsesGoBackToIdle(t *testing.T) {
	h := newBlockSyncHarness()
	finishedState := h.factory.CreateFinishedCARState(nil)
	shouldBeIdleState := finishedState.processState(h.ctx)

	require.IsType(t, &idleState{}, shouldBeIdleState, "next state should be idle")
}

func TestFinishedWithResponsesMoveToWaitingForChunk(t *testing.T) {
	response := builders.BlockAvailabilityResponseInput().Build().Message
	h := newBlockSyncHarness()
	finishedState := h.factory.CreateFinishedCARState([]*gossipmessages.BlockAvailabilityResponseMessage{response})
	shouldBeWaitingState := finishedState.processState(h.ctx)

	require.IsType(t, &waitingForChunksState{}, shouldBeWaitingState, "next state should be waiting for chunk")
}

func TestFinishedContextTerminationFlow(t *testing.T) {
	h := newBlockSyncHarness()
	response := builders.BlockAvailabilityResponseInput().Build().Message
	finishedState := h.factory.CreateFinishedCARState([]*gossipmessages.BlockAvailabilityResponseMessage{response})

	h.cancel()
	shouldBeNil := finishedState.processState(h.ctx)

	require.Nil(t, shouldBeNil, "context terminated, state should be nil")
}

func TestFinishedNOP(t *testing.T) {
	h := newBlockSyncHarness()
	finishedState := h.factory.CreateFinishedCARState([]*gossipmessages.BlockAvailabilityResponseMessage{})

	// sanity test, these should do nothing
	finishedState.gotBlocks(h.ctx, nil)
	finishedState.blockCommitted(h.ctx)
	finishedState.gotAvailabilityResponse(h.ctx, nil)
}

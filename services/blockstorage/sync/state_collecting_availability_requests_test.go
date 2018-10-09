package sync

import (
	"errors"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCollectingAvailabilityResponsesReturnsToIdleOnGossipError(t *testing.T) {
	h := newBlockSyncHarness()

	h.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(10)).Times(1)
	h.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any).Return(nil, errors.New("gossip failure")).Times(1)

	collectingState := h.sf.CreateCollectingAvailabilityResponseState()
	nextShouldBeIdle := collectingState.processState(h.ctx)

	_, isIdle := nextShouldBeIdle.(*idleState)

	require.True(t, isIdle, "should be idle on gossip error")

	h.verifyMocks(t)
}

func TestCollectingAvailabilityResponsesMovesToFinishedCollecting(t *testing.T) {
	h := newBlockSyncHarness() //.withCollectResponseTimeout(1 * time.Millisecond)

	h.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(10)).Times(1)
	h.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any).Return(nil, nil).Times(1)

	collectingState := h.sf.CreateCollectingAvailabilityResponseState()
	nextShouldBeFinished := collectingState.processState(h.ctx)

	_, isIdle := nextShouldBeFinished.(*finishedCARState)

	require.True(t, isIdle, "state transition incorrect")

	h.verifyMocks(t)
}

func TestCollectingAvailabilityResponsesAddsAResponse(t *testing.T) {
	h := newBlockSyncHarness() //.withCollectResponseTimeout(1 * time.Millisecond)

	collectingState := h.sf.CreateCollectingAvailabilityResponseState()
	message := builders.BlockAvailabilityResponseInput().Build().Message
	collectingState.gotAvailabilityResponse(message)
	cs := collectingState.(*collectingAvailabilityResponsesState)
	require.True(t, len(cs.responses) == 1, "should have 1 response after adding it")
	require.Equal(t, message.Sender, cs.responses[0].Sender, "state sender should match message sender")
	require.Equal(t, message.SignedBatchRange, cs.responses[0].SignedBatchRange, "state payload should match message")
}

func TestCollectingAvailabilityContextTermination(t *testing.T) {
	h := newBlockSyncHarness() //.withCollectResponseTimeout(1 * time.Millisecond)
	h.cancel()

	h.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(10)).Times(1)
	h.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any).Return(nil, nil).Times(1)

	collectingState := h.sf.CreateCollectingAvailabilityResponseState()
	nextState := collectingState.processState(h.ctx)

	require.Nil(t, nextState, "context terminated, next state should be nil")

	h.verifyMocks(t)
}

func TestCollectingAvailabilityResponsesNOP(t *testing.T) {
	h := newBlockSyncHarness()
	car := h.sf.CreateCollectingAvailabilityResponseState()
	// these calls should do nothing, this is just a sanity that they do not panic and return nothing
	car.gotBlocks(nil)
	car.blockCommitted(primitives.BlockHeight(0))
}

package sync

import (
	"errors"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
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
	h := newBlockSyncHarness().WithCollectResponseTimeout(1 * time.Millisecond)

	h.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(10)).Times(1)
	h.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any).Return(nil, nil).Times(1)

	collectingState := h.sf.CreateCollectingAvailabilityResponseState()
	nextShouldBeIdle := collectingState.processState(h.ctx)

	_, isIdle := nextShouldBeIdle.(*finishedCARState)

	require.True(t, isIdle, "state transition incorrect")

	h.verifyMocks(t)
}

func TestCollectingAvailabilityResponsesNOP(t *testing.T) {
	h := newBlockSyncHarness()
	car := h.sf.CreateCollectingAvailabilityResponseState()
	// these calls should do nothing, this is just a sanity that they do not panic and return nothing
	car.gotBlocks(h.config.NodePublicKey(), nil)
	car.blockCommitted(primitives.BlockHeight(0))
}

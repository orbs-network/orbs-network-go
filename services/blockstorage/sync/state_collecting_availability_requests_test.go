package sync

import (
	"errors"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCollectingAvailabilityResponsesReturnsToIdleOnGossipError(t *testing.T) {
	h := newBlockSyncHarness()

	h.storage.When("UpdateConsensusAlgosAboutLatestCommittedBlock", mock.Any).Times(1)
	h.expectLastCommittedBlockHeight(primitives.BlockHeight(10))
	h.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).Return(nil, errors.New("gossip failure")).Times(1)

	collectingState := h.sf.CreateCollectingAvailabilityResponseState()
	nextShouldBeIdle := collectingState.processState(h.ctx)

	require.IsType(t, &idleState{}, nextShouldBeIdle, "should be idle on gossip error")

	h.verifyMocks(t)
}

func TestCollectingAvailabilityResponsesReturnsToIdleOnInvalidRequestSize(t *testing.T) {
	// this can probably happen only if BatchSize config is invalid
	h := newBlockSyncHarness().withBatchSize(0)

	h.storage.When("UpdateConsensusAlgosAboutLatestCommittedBlock", mock.Any).Times(1)
	h.expectLastCommittedBlockHeight(primitives.BlockHeight(0)) // new server

	collectingState := h.sf.CreateCollectingAvailabilityResponseState()
	nextShouldBeIdle := collectingState.processState(h.ctx)

	require.IsType(t, &idleState{}, nextShouldBeIdle, "should be idle on gossip flow error")

	h.verifyMocks(t)
}

func TestCollectingAvailabilityResponsesMovesToFinishedCollecting(t *testing.T) {
	h := newBlockSyncHarness()

	h.storage.When("UpdateConsensusAlgosAboutLatestCommittedBlock", mock.Any).Times(1)
	h.expectLastCommittedBlockHeight(primitives.BlockHeight(10))
	h.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).Return(nil, nil).Times(1)

	message := builders.BlockAvailabilityResponseInput().Build().Message
	collectingState := h.sf.CreateCollectingAvailabilityResponseState()
	nextShouldBeFinished := h.nextState(collectingState, func() {
		require.NoError(t, test.EventuallyVerify(10*time.Millisecond, h.gossip), "broadcast was not sent out")
		collectingState.gotAvailabilityResponse(h.ctx, message)
	})

	require.IsType(t, &finishedCARState{}, nextShouldBeFinished, "state should transition to finished CAR")
	fcar := nextShouldBeFinished.(*finishedCARState)
	require.Equal(t, 1, len(fcar.responses), "there should be one response")
	require.Equal(t, message.Sender, fcar.responses[0].Sender, "state sender should match message sender")
	require.Equal(t, message.SignedBatchRange, fcar.responses[0].SignedBatchRange, "state payload should match message")

	h.verifyMocks(t)
}

func TestCollectingAvailabilityContextTermination(t *testing.T) {
	h := newBlockSyncHarness()
	h.cancel()

	h.storage.When("UpdateConsensusAlgosAboutLatestCommittedBlock", mock.Any).Times(1)
	h.expectLastCommittedBlockHeight(primitives.BlockHeight(10))
	h.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).Return(nil, nil).Times(1)

	collectingState := h.sf.CreateCollectingAvailabilityResponseState()
	nextState := collectingState.processState(h.ctx)

	require.Nil(t, nextState, "context terminated, next state should be nil")

	h.verifyMocks(t)
}

func TestCollectingReceiveResponseWhenNotReadyDoesNotBlock(t *testing.T) {
	h := newBlockSyncHarness()
	h = h.withCtxTimeout(h.config.collectResponses / 2)

	collectingState := h.sf.CreateCollectingAvailabilityResponseState()
	// not calling the process state will not activate the reader part
	message := builders.BlockAvailabilityResponseInput().Build().Message
	collectingState.gotAvailabilityResponse(h.ctx, message) // this will block if the test fails
	h.cancel()
}

func TestCollectingAvailabilityResponsesNOP(t *testing.T) {
	h := newBlockSyncHarness()
	car := h.sf.CreateCollectingAvailabilityResponseState()
	// these calls should do nothing, this is just a sanity that they do not panic and return nothing
	blockmessage := builders.BlockSyncResponseInput().Build().Message
	car.gotBlocks(h.ctx, blockmessage)
	car.blockCommitted(h.ctx)
}

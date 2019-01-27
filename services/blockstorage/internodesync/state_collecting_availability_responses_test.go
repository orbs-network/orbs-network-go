package internodesync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStateCollectingAvailabilityResponses_ReturnsToIdleOnGossipError(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness()

		h.expectPreSynchronizationUpdateOfConsensusAlgos(10)
		h.expectBroadcastOfBlockAvailabilityRequestToFail()

		state := h.factory.CreateCollectingAvailabilityResponseState()
		nextState := state.processState(ctx)

		require.IsType(t, &idleState{}, nextState, "next state should be idle on gossip error")
		h.verifyMocks(t)
	})
}

func TestStateCollectingAvailabilityResponses_ReturnsToIdleOnInvalidRequestSizeConfig(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		// this can probably happen only if BatchSize config is invalid
		h := newBlockSyncHarness().withBatchSize(0)

		h.expectPreSynchronizationUpdateOfConsensusAlgos(0) // new server

		state := h.factory.CreateCollectingAvailabilityResponseState()
		nextState := state.processState(ctx)

		require.IsType(t, &idleState{}, nextState, "next state should be idle on gossip error flow")
		h.verifyMocks(t)
	})
}

func TestStateCollectingAvailabilityResponses_MovesToFinishedCollecting(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		manualCollectResponsesTimer := synchronization.NewTimerWithManualTick()
		h := newBlockSyncHarnessWithCollectResponsesTimer(func() *synchronization.Timer {
			return manualCollectResponsesTimer
		})

		h.expectPreSynchronizationUpdateOfConsensusAlgos(10)
		h.expectBroadcastOfBlockAvailabilityRequest()

		message := builders.BlockAvailabilityResponseInput().Build().Message
		state := h.factory.CreateCollectingAvailabilityResponseState()
		nextState := h.processStateInBackgroundAndWaitUntilFinished(ctx, state, func() {
			h.verifyBroadcastOfBlockAvailabilityRequest(t)
			h.factory.conduit <- message
			manualCollectResponsesTimer.ManualTick()
		})

		require.IsType(t, &finishedCARState{}, nextState, "state should transition to finished CAR")
		fcar := nextState.(*finishedCARState)
		require.Equal(t, 1, len(fcar.responses), "there should be one response")
		require.Equal(t, message.Sender, fcar.responses[0].Sender, "state sender should match message sender")
		require.Equal(t, message.SignedBatchRange, fcar.responses[0].SignedBatchRange, "state payload should match message")

		h.verifyMocks(t)
	})
}

func TestStateCollectingAvailabilityResponses_ContextTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h := newBlockSyncHarness()

	h.expectPreSynchronizationUpdateOfConsensusAlgos(10)
	h.expectBroadcastOfBlockAvailabilityRequest()

	state := h.factory.CreateCollectingAvailabilityResponseState()
	nextState := state.processState(ctx)

	require.Nil(t, nextState, "context terminated, next state should be nil")

	h.verifyMocks(t)
}

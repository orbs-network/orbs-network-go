package sync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestStateCollectingAvailabilityResponses_ReturnsToIdleOnGossipError(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness()

		h.expectPreSynchronizationUpdateOfConsensusAlgos(10)
		h.expectBroadcastOfBlockAvailabilityRequestToFail()

		collectingState := h.factory.CreateCollectingAvailabilityResponseState()
		nextShouldBeIdle := collectingState.processState(ctx)

		require.IsType(t, &idleState{}, nextShouldBeIdle, "should be idle on gossip error")

		h.verifyMocks(t)
	})
}

func TestStateCollectingAvailabilityResponses_ReturnsToIdleOnInvalidRequestSize(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		// this can probably happen only if BatchSize config is invalid
		h := newBlockSyncHarness().withBatchSize(0)

		h.expectPreSynchronizationUpdateOfConsensusAlgos(0) // new server

		collectingState := h.factory.CreateCollectingAvailabilityResponseState()
		nextShouldBeIdle := collectingState.processState(ctx)

		require.IsType(t, &idleState{}, nextShouldBeIdle, "should be idle on gossip flow error")

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
		collectingState := h.factory.CreateCollectingAvailabilityResponseState()
		nextState := h.processStateAndWaitUntilFinished(ctx, collectingState, func() {
			require.NoError(t, test.EventuallyVerify(10*time.Millisecond, h.gossip), "broadcast was not sent out")
			collectingState.gotAvailabilityResponse(ctx, message)
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

	collectingState := h.factory.CreateCollectingAvailabilityResponseState()
	nextState := collectingState.processState(ctx)

	require.Nil(t, nextState, "context terminated, next state should be nil")

	h.verifyMocks(t)
}

func TestStateCollectingAvailabilityResponses_ReceiveResponseWhenNotReadyDoesNotBlock(t *testing.T) {
	h := newBlockSyncHarness()
	test.WithContextWithTimeout(h.config.collectResponses/2, func(ctx context.Context) {

		collectingState := h.factory.CreateCollectingAvailabilityResponseState()
		// not calling the process state will not activate the reader part
		message := builders.BlockAvailabilityResponseInput().Build().Message
		collectingState.gotAvailabilityResponse(ctx, message) // this will block if the test fails
	})
}

func TestStateCollectingAvailabilityResponses_NOP(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness()
		car := h.factory.CreateCollectingAvailabilityResponseState()
		// these calls should do nothing, this is just a sanity that they do not panic and return nothing
		blockmessage := builders.BlockSyncResponseInput().Build().Message
		car.gotBlocks(ctx, blockmessage)
		car.blockCommitted(ctx)
	})

}

package internodesync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStateFinishedCollectingAvailabilityResponses_ReturnsToIdleWhenNoResponsesReceived(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness()

		state := h.factory.CreateFinishedCARState(nil)
		nextState := state.processState(ctx)

		require.IsType(t, &idleState{}, nextState, "next state should be idle")
	})
}

func TestStateFinishedCollectingAvailabilityResponses_MovesToWaitingForChunks(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness()

		response := builders.BlockAvailabilityResponseInput().Build().Message
		state := h.factory.CreateFinishedCARState([]*gossipmessages.BlockAvailabilityResponseMessage{response})
		nextState := state.processState(ctx)

		require.IsType(t, &waitingForChunksState{}, nextState, "next state should be waiting for chunks")
	})
}

func TestStateFinishedCollectingAvailabilityResponses_ContextTerminationFlow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	h := newBlockSyncHarness()

	response := builders.BlockAvailabilityResponseInput().Build().Message
	state := h.factory.CreateFinishedCARState([]*gossipmessages.BlockAvailabilityResponseMessage{response})

	cancel()
	shouldBeNil := state.processState(ctx)

	require.Nil(t, shouldBeNil, "context terminated, state should be nil")
}

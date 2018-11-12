package externalsync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFinishedWithNoResponsesGoBackToIdle(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness()
		finishedState := h.factory.CreateFinishedCARState(nil)
		shouldBeIdleState := finishedState.processState(ctx)

		require.IsType(t, &idleState{}, shouldBeIdleState, "next state should be idle")
	})
}

func TestFinishedWithResponsesMoveToWaitingForChunk(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		response := builders.BlockAvailabilityResponseInput().Build().Message
		h := newBlockSyncHarness()
		finishedState := h.factory.CreateFinishedCARState([]*gossipmessages.BlockAvailabilityResponseMessage{response})
		shouldBeWaitingState := finishedState.processState(ctx)

		require.IsType(t, &waitingForChunksState{}, shouldBeWaitingState, "next state should be waiting for chunk")
	})
}

func TestFinishedContextTerminationFlow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	h := newBlockSyncHarness()
	response := builders.BlockAvailabilityResponseInput().Build().Message
	finishedState := h.factory.CreateFinishedCARState([]*gossipmessages.BlockAvailabilityResponseMessage{response})

	cancel()
	shouldBeNil := finishedState.processState(ctx)

	require.Nil(t, shouldBeNil, "context terminated, state should be nil")
}

func TestFinishedNOP(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newBlockSyncHarness()
		finishedState := h.factory.CreateFinishedCARState([]*gossipmessages.BlockAvailabilityResponseMessage{})

		// sanity test, these should do nothing
		finishedState.gotBlocks(ctx, nil)
		finishedState.blockCommitted(ctx)
		finishedState.gotAvailabilityResponse(ctx, nil)
	})
}

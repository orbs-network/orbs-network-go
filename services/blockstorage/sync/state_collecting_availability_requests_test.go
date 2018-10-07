package sync

import (
	"errors"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCollectingAvailabilityResponsesReturnsToIdleOnGossipError(t *testing.T) {
	h := newBlockSyncHarness()

	//harness.storage.When("UpdateConsensusAlgosAboutLatestCommittedBlock").Return().Times(1)
	h.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(10)).Times(1)
	h.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any).Return(nil, errors.New("gossip failure")).Times(1)

	collectingState := h.sf.CreateCollectingAvailabilityResponseState()
	nextShouldBeIdle := collectingState.processState(h.ctx)

	_, isIdle := nextShouldBeIdle.(*idleState)

	require.True(t, isIdle, "should be idle on gossip error")

	h.verifyMocks(t)
}

package sync

import (
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestBlockSyncStartsWithImmediateSync(t *testing.T) {
	h := newBlockSyncHarness().withNoCommitTimeout(time.Hour) // we want to see that the sync immediately starts, and not in an hour

	h.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(10)).Times(1)
	h.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).Return(nil, nil).Times(1)

	sync := NewBlockSync(h.ctx, h.config, h.gossip, h.storage, h.logger)

	h.eventuallyVerifyMocks(t, 2) // just need to verify we used gossip/storage for sync
	h.cancel()
	shutdown := h.waitForShutdown(sync)
	require.True(t, shutdown, "expecting state to be set to nil (=shutdown)")
}

func TestBlockSyncStaysInIdleOnBlockCommitExternalMessage(t *testing.T) {
	// although we test this use case at the service level, this test is testing the same logic on the sync unit level
	// its to cover that specific line of code in blockSync engine, rather then the service handler code
	// (or the idle state code)

	h := newBlockSyncHarness().withNoCommitTimeout(8 * time.Millisecond)
	h.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(10)).Times(1)
	h.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).Return(nil, nil).Times(1) // only one allowed

	sync := NewBlockSync(h.ctx, h.config, h.gossip, h.storage, h.logger)
	idleReached := h.waitForState(sync, h.sf.CreateIdleState())
	require.True(t, idleReached, "idle state was not reached when expected, most likely something else is broken")

	// "commit" blocks at a rate of 1/ms, do not assume anything about the implementation
	for i := 1; i < 10; i++ {
		sync.HandleBlockCommitted()
		time.Sleep(500 * time.Microsecond)
		require.IsType(t, &idleState{}, sync.currentState, "state should remain idle")
	}

	h.cancel() // kill the sync (goroutine)
}

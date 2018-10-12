package sync

import (
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestBlockSyncShutdown(t *testing.T) {
	h := newBlockSyncHarness()

	h.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(10)).Times(1)
	h.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any).Return(nil, nil).Times(1)
	sync := NewBlockSync(h.ctx, h.config, h.gossip, h.storage, h.logger)
	h.cancel()
	time.Sleep(time.Millisecond) // waiting for the sync to start
	// TODO: refactor this once more logic is added, this is not really checking the the goroutine stopped
	require.True(t, sync.terminated, "expecting the stop flag up")
}

func TestBlockSyncStaysInIdleOnBlockCommitExternalMessage(t *testing.T) {
	// although we test this use case at the service level, this test is testing the same logic on the sync unit level
	// its to cover that specific line of code in blockSync engine, rather then the service handler code
	// (or the idle state code)

	h := newBlockSyncHarness().withNoCommitTimeout(3 * time.Millisecond)
	h.gossip.Never("BroadcastBlockAvailabilityRequest")

	sync := NewBlockSync(h.ctx, h.config, h.gossip, h.storage, h.logger)
	time.Sleep(time.Millisecond) // give the sync time to start

	// "commit" blocks at a rate of 1/ms, do not assume anything about the implementation
	for i := 1; i < 10; i++ {
		sync.HandleBlockCommitted()
		time.Sleep(time.Millisecond)
		require.IsType(t, &idleState{}, sync.currentState, "state should remain idle")
	}

	h.cancel() // kill the sync (goroutine)
}

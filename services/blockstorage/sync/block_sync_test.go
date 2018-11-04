package sync

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestBlockSyncStartsWithImmediateSync(t *testing.T) {
	h := newBlockSyncHarness().withNoCommitTimeout(time.Hour) // we want to see that the sync immediately starts, and not in an hour

	h.expectingSyncOnStart()

	sync := NewBlockSync(h.ctx, h.config, h.gossip, h.storage, h.logger, h.m)

	h.eventuallyVerifyMocks(t, 2) // just need to verify we used gossip/storage for sync
	h.cancel()
	shutdown := h.waitForShutdown(sync)
	require.True(t, shutdown, "expecting state to be set to nil (=shutdown)")
}

func TestBlockSyncStaysInIdleOnBlockCommitExternalMessage(t *testing.T) {
	// although we test this use case at the service level, this test is testing the same logic on the sync unit level
	// its to cover that specific line of code in blockSync engine, rather then the service handler code
	// (or the idle state code)

	h := newBlockSyncHarness().withNoCommitTimeout(20 * time.Millisecond)
	h.expectingSyncOnStart()

	sync := NewBlockSync(h.ctx, h.config, h.gossip, h.storage, h.logger, h.m)
	idleReached := h.waitForState(sync, h.sf.CreateIdleState())
	require.True(t, idleReached, "idle state was not reached when expected, most likely something else is broken")

	// "commit" blocks loop
	go func() {
		for {
			sync.HandleBlockCommitted(h.ctx)
			time.Sleep(1 * time.Millisecond)
		}
	}()

	time.Sleep(30 * time.Millisecond)

	require.IsType(t, &idleState{}, sync.currentState, "state should remain idle")

	h.verifyMocks(t)
	h.cancel() // kill the sync (goroutine)
}

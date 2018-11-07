package sync

import (
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBlockSyncStartsWithImmediateSync(t *testing.T) {
	manualNoCommitTimer := synchronization.NewTimerWithManualTick()
	h := newBlockSyncHarnessWithManualNoCommitTimeoutTimer(manualNoCommitTimer)

	h.expectingSyncOnStart()

	sync := NewBlockSync(h.ctx, h.config, h.gossip, h.storage, h.logger, h.metricFactory)

	h.eventuallyVerifyMocks(t, 2) // just need to verify we used gossip/storage for sync
	h.cancel()
	shutdown := h.waitForShutdown(sync)
	require.True(t, shutdown, "expecting state to be set to nil (=shutdown)")
}

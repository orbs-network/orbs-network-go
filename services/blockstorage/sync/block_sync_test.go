package sync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBlockSyncStartsWithImmediateSync(t *testing.T) {
	manualNoCommitTimer := synchronization.NewTimerWithManualTick()
	h := newBlockSyncHarnessWithManualNoCommitTimeoutTimer(manualNoCommitTimer)
	var bs *BlockSync

	test.WithContext(func(ctx context.Context) {
		h.expectingSyncOnStart()

		bs = NewBlockSync(ctx, h.config, h.gossip, h.storage, h.logger, h.metricFactory)

		h.eventuallyVerifyMocks(t, 2) // just need to verify we used gossip/storage for sync
	})

	shutdown := h.waitForShutdown(bs)
	require.True(t, shutdown, "expecting state to be set to nil (=shutdown)")
}

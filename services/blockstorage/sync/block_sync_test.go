package sync

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBlockSyncShutdown(t *testing.T) {
	h := newBlockSyncHarness()
	sync := NewBlockSync(h.ctx, h.config, h.gossip, h.storage)
	sync.Shutdown()
	// TODO: refactor this once more logic is added, this is not really checking the the goroutine stopped
	require.True(t, sync.shouldStop, "expecting the stop flag up")
}

package sync

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNewBlockSync(t *testing.T) {
	h := primitives.BlockHeight(10)
	sync := NewBlockSync(h, 3*time.Millisecond)
	require.NotNil(t, sync, "block sync initialized")
	require.EqualValues(t, h, sync.lastBlockHeight, "block sync known height initialized correctly")
	sync.Shutdown()
}

func TestBlockSyncShutdown(t *testing.T) {
	h := primitives.BlockHeight(10)
	sync := NewBlockSync(h, 3*time.Millisecond)
	sync.Shutdown()
	// TODO: refactor this once more logic is added, this is not really checking the the goroutine stopped
	require.True(t, sync.shouldStop, "expecting the stop flag up")
}

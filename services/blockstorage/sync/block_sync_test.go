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
	time.Sleep(time.Millisecond)
	// TODO: refactor this once more logic is added, this is not really checking the the goroutine stopped
	require.True(t, sync.terminated, "expecting the stop flag up")
}

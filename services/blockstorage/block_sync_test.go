package blockstorage

import (
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/stretchr/testify/require"
	"testing"
)

type blockSyncHarness struct {
	blockSync *BlockSync
	gossip    *gossiptopics.MockBlockSync
	storage   *blockSyncStorageMock
}

func newBlockSyncHarness() *blockSyncHarness {
	cfg := config.EmptyConfig()
	gossip := &gossiptopics.MockBlockSync{}
	storage := &blockSyncStorageMock{}

	blockSync := &BlockSync{
		reporting: log.GetLogger(),
		config:    cfg,
		storage:   storage,
		gossip:    gossip,
		events:    nil,
	}

	return &blockSyncHarness{
		blockSync: blockSync,
		gossip:    gossip,
		storage:   storage,
	}
}

func (h *blockSyncHarness) verifyMocks(t *testing.T) {
	ok, err := mock.VerifyMocks(h.storage, h.gossip)
	require.NoError(t, err)
	require.True(t, ok)

}

func TestTransitionFromSyncStartToCollectAvailabilityResponses(t *testing.T) {
	harness := newBlockSyncHarness()

	var event interface{}
	var responses []*gossipmessages.BlockAvailabilityResponseMessage

	harness.storage.When("UpdateConsensusAlgosAboutLatestCommittedBlock").Return().Times(1)
	harness.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(10)).Times(1)
	harness.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any).Return(nil, nil).Times(1)

	newState, responses := harness.blockSync.transitionState(BLOCK_SYNC_STATE_START_SYNC, event, responses)

	require.Equal(t, BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES, newState)
	require.Empty(t, responses, "no responses were sent yet")

	harness.verifyMocks(t)
}

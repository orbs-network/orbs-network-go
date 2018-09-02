package blockstorage

import (
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

type blockSyncHarness struct {
	blockSync                  *BlockSync
	gossip                     *gossiptopics.MockBlockSync
	storage                    *blockSyncStorageMock
	collectAvailabilityTrigger *synchronization.PeriodicalTriggerMock
}

func newBlockSyncHarness() *blockSyncHarness {
	cfg := config.EmptyConfig()
	gossip := &gossiptopics.MockBlockSync{}
	storage := &blockSyncStorageMock{}
	collectAvailabilityTrigger := &synchronization.PeriodicalTriggerMock{}

	blockSync := &BlockSync{
		reporting: log.GetLogger(),
		config:    cfg,
		storage:   storage,
		gossip:    gossip,
		events:    nil,
	}

	return &blockSyncHarness{
		blockSync:                  blockSync,
		gossip:                     gossip,
		storage:                    storage,
		collectAvailabilityTrigger: collectAvailabilityTrigger,
	}
}

func (h *blockSyncHarness) verifyMocks(t *testing.T) {
	ok, err := mock.VerifyMocks(h.storage, h.gossip, h.collectAvailabilityTrigger)
	require.NoError(t, err)
	require.True(t, ok)

}

func TestStartSyncHappyFlow(t *testing.T) {
	harness := newBlockSyncHarness()

	event := startSyncEvent{}
	availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}

	harness.storage.When("UpdateConsensusAlgosAboutLatestCommittedBlock").Return().Times(1)
	harness.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(10)).Times(1)
	harness.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any).Return(nil, nil).Times(1)
	harness.collectAvailabilityTrigger.When("Reset").Return().Times(1)

	newState, availabilityResponses := harness.blockSync.transitionState(BLOCK_SYNC_STATE_IDLE, event, availabilityResponses, harness.collectAvailabilityTrigger)

	require.Equal(t, BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES, newState)
	require.Empty(t, availabilityResponses, "no availabilityResponses were sent yet")

	harness.verifyMocks(t)
}

func TestIdleIgnoresInvalidEvents(t *testing.T) {
	harness := newBlockSyncHarness()

	event := collectingAvailabilityFinishedEvent{}
	availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}

	newState, availabilityResponses := harness.blockSync.transitionState(BLOCK_SYNC_STATE_IDLE, event, availabilityResponses, harness.collectAvailabilityTrigger)

	require.Equal(t, BLOCK_SYNC_STATE_IDLE, newState)
	require.NotEmpty(t, availabilityResponses, "availabilityResponses were sent but shouldn't have")

	harness.verifyMocks(t)
}

func TestStartSyncGossipFailure(t *testing.T) {
	harness := newBlockSyncHarness()

	event := startSyncEvent{}
	availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}

	harness.storage.When("UpdateConsensusAlgosAboutLatestCommittedBlock").Return().Times(1)
	harness.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(10)).Times(1)
	harness.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any).Return(nil, errors.New("gossip failure")).Times(1)

	newState, availabilityResponses := harness.blockSync.transitionState(BLOCK_SYNC_STATE_IDLE, event, availabilityResponses, harness.collectAvailabilityTrigger)

	require.Equal(t, BLOCK_SYNC_STATE_IDLE, newState)
	require.NotEmpty(t, availabilityResponses, "availabilityResponses were sent but shouldn't have")

	harness.verifyMocks(t)
}

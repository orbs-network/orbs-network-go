package blockstorage

import (
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"reflect"
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

func typeOfEvent(event interface{}) string {
	return reflect.TypeOf(event).String()
}

func allEventsExcept(eventTypes ...string) (res []interface{}) {
	allEvents := []interface{}{
		startSyncEvent{},
		collectingAvailabilityFinishedEvent{},
		builders.BlockAvailabilityResponseInput().Build().Message,
	}

	res = []interface{}{}

	for _, event := range allEvents {
		shouldAdd := true
		for _, eventTypeToRemove := range eventTypes {
			if typeOfEvent(event) == eventTypeToRemove {
				shouldAdd = false
				break
			}
		}

		if shouldAdd {
			res = append(res, event)
		}
	}
	return
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

	require.Equal(t, BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES, newState, "state change does not match expected")
	require.Empty(t, availabilityResponses, "no availabilityResponses were sent yet")

	harness.verifyMocks(t)
}

func TestIdleIgnoresInvalidEvents(t *testing.T) {
	events := allEventsExcept("blockstorage.startSyncEvent")

	for _, event := range events {
		t.Run(typeOfEvent(event), func(t *testing.T) {
			harness := newBlockSyncHarness()

			availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}

			newState, availabilityResponses := harness.blockSync.transitionState(BLOCK_SYNC_STATE_IDLE, event, availabilityResponses, harness.collectAvailabilityTrigger)

			require.Equal(t, BLOCK_SYNC_STATE_IDLE, newState, "state change does not match expected")
			require.NotEmpty(t, availabilityResponses, "availabilityResponses were sent but shouldn't have")

			harness.verifyMocks(t)
		})
	}
}

func TestStartSyncGossipFailure(t *testing.T) {
	harness := newBlockSyncHarness()

	event := startSyncEvent{}
	availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}

	harness.storage.When("UpdateConsensusAlgosAboutLatestCommittedBlock").Return().Times(1)
	harness.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(10)).Times(1)
	harness.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any).Return(nil, errors.New("gossip failure")).Times(1)

	newState, availabilityResponses := harness.blockSync.transitionState(BLOCK_SYNC_STATE_IDLE, event, availabilityResponses, harness.collectAvailabilityTrigger)

	require.Equal(t, BLOCK_SYNC_STATE_IDLE, newState, "state change does not match expected")
	require.NotEmpty(t, availabilityResponses, "availabilityResponses were sent but shouldn't have")

	harness.verifyMocks(t)
}

func TestCollectingAvailabilityNoResponsesFlow(t *testing.T) {
	harness := newBlockSyncHarness()

	event := collectingAvailabilityFinishedEvent{}
	availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{}

	newState, availabilityResponses := harness.blockSync.transitionState(BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES, event, availabilityResponses, harness.collectAvailabilityTrigger)

	require.Equal(t, BLOCK_SYNC_STATE_IDLE, newState, "state change does not match expected")
	require.Empty(t, availabilityResponses, "no availabilityResponses should have been received")

	harness.verifyMocks(t)
}

func TestCollectingAvailabilityAddingResponseFlow(t *testing.T) {
	harness := newBlockSyncHarness()

	event := builders.BlockAvailabilityResponseInput().Build().Message
	availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{nil}

	newState, availabilityResponses := harness.blockSync.transitionState(BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES, event, availabilityResponses, harness.collectAvailabilityTrigger)

	require.Equal(t, BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES, newState, "state change does not match expected")
	require.Equal(t, availabilityResponses, []*gossipmessages.BlockAvailabilityResponseMessage{nil, event}, "availabilityResponses should have the event added")

	harness.verifyMocks(t)
}

func TestCollectingAvailabilityIgnoresInvalidEvents(t *testing.T) {

	events := allEventsExcept("blockstorage.collectingAvailabilityFinishedEvent", "*gossipmessages.BlockAvailabilityResponseMessage")
	for _, event := range events {
		t.Run(typeOfEvent(event), func(t *testing.T) {
			harness := newBlockSyncHarness()

			availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}

			newState, availabilityResponses := harness.blockSync.transitionState(BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES, event, availabilityResponses, harness.collectAvailabilityTrigger)

			require.Equal(t, BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES, newState, "state change does not match expected")
			require.Equal(t, availabilityResponses, []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}, "availabilityResponses should remain the same")

			harness.verifyMocks(t)
		})
	}
}

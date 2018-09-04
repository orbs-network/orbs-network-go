package blockstorage

import (
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"reflect"
	"strconv"
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
		builders.BlockAvailabilityRequestInput().Build().Message,
		builders.BlockSyncRequestInput().Build().Message,
		builders.BlockSyncResponseInput().Build().Message,
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

func allStates() []blockSyncState {
	return []blockSyncState{
		BLOCK_SYNC_STATE_IDLE,
		BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES,
		BLOCK_SYNC_PETITIONER_WAITING_FOR_CHUNK,
	}
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
	events := allEventsExcept(
		"blockstorage.startSyncEvent",
		"*gossipmessages.BlockAvailabilityRequestMessage",
		"*gossipmessages.BlockSyncRequestMessage")

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
	events := allEventsExcept(
		"blockstorage.collectingAvailabilityFinishedEvent",
		"*gossipmessages.BlockAvailabilityResponseMessage",
		"*gossipmessages.BlockAvailabilityRequestMessage",
		"*gossipmessages.BlockSyncRequestMessage")

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

func TestCollectingAvailabilitySendsBlockSyncRequest(t *testing.T) {
	harness := newBlockSyncHarness()

	harness.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(0)).Times(1)
	harness.gossip.When("SendBlockSyncRequest", mock.Any).Return(nil, nil).Times(1)

	event := collectingAvailabilityFinishedEvent{}
	availabilityResponse := builders.BlockAvailabilityResponseInput().Build().Message
	availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{availabilityResponse}

	newState, availabilityResponses := harness.blockSync.transitionState(BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES, event, availabilityResponses, harness.collectAvailabilityTrigger)

	require.Equal(t, BLOCK_SYNC_PETITIONER_WAITING_FOR_CHUNK, newState, "state change does not match expected")
	require.Equal(t, availabilityResponses, []*gossipmessages.BlockAvailabilityResponseMessage{availabilityResponse}, "availabilityResponses should not have been changed")

	harness.verifyMocks(t)
}

func TestCollectingAvailabilityDoesNothingIfNoResponsesCollected(t *testing.T) {
	harness := newBlockSyncHarness()

	event := collectingAvailabilityFinishedEvent{}
	availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{}

	newState, availabilityResponses := harness.blockSync.transitionState(BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES, event, availabilityResponses, harness.collectAvailabilityTrigger)

	require.Equal(t, BLOCK_SYNC_STATE_IDLE, newState, "state change does not match expected")
	require.Empty(t, availabilityResponses, "availabilityResponses should not have been changed")

	harness.verifyMocks(t)
}

func TestCollectingAvailabilityDoesNothingIfFailsToSendRequest(t *testing.T) {
	harness := newBlockSyncHarness()

	harness.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(0)).Times(1)
	harness.gossip.When("SendBlockSyncRequest", mock.Any).Return(nil, errors.New("gossip failure")).Times(1)

	event := collectingAvailabilityFinishedEvent{}
	availabilityResponse := builders.BlockAvailabilityResponseInput().Build().Message
	availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{availabilityResponse}

	newState, availabilityResponses := harness.blockSync.transitionState(BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES, event, availabilityResponses, harness.collectAvailabilityTrigger)

	require.Equal(t, BLOCK_SYNC_STATE_IDLE, newState, "state change does not match expected")
	require.Equal(t, availabilityResponses, []*gossipmessages.BlockAvailabilityResponseMessage{availabilityResponse}, "availabilityResponses should not have been changed")

	harness.verifyMocks(t)
}

func TestWaitingForChunk(t *testing.T) {
	harness := newBlockSyncHarness()

	harness.storage.When("ValidateBlockForCommit", mock.Any).Return(nil, nil).Times(91)
	harness.storage.When("CommitBlock", mock.Any).Return(nil, nil).Times(91)

	event := builders.BlockSyncResponseInput().Build().Message
	availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}

	newState, availabilityResponses := harness.blockSync.transitionState(BLOCK_SYNC_PETITIONER_WAITING_FOR_CHUNK, event, availabilityResponses, harness.collectAvailabilityTrigger)
	require.Equal(t, BLOCK_SYNC_STATE_IDLE, newState, "state change does not match expected")
	require.Equal(t, availabilityResponses, []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}, "availabilityResponses should not have been changed")

	harness.verifyMocks(t)
}

func TestWaitingForChunkBlockValidationFailed(t *testing.T) {
	harness := newBlockSyncHarness()

	event := builders.BlockSyncResponseInput().Build().Message
	availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}

	harness.storage.When("ValidateBlockForCommit", mock.Any).Call(func(input *services.ValidateBlockForCommitInput) error {
		if input.BlockPair.ResultsBlock.Header.BlockHeight().Equal(event.SignedChunkRange.FirstBlockHeight() + 50) {
			return errors.New("failed to validate block #51")
		}
		return nil
	}).Times(51)
	harness.storage.When("CommitBlock", mock.Any).Return(nil, nil).Times(50)

	newState, availabilityResponses := harness.blockSync.transitionState(BLOCK_SYNC_PETITIONER_WAITING_FOR_CHUNK, event, availabilityResponses, harness.collectAvailabilityTrigger)
	require.Equal(t, BLOCK_SYNC_STATE_IDLE, newState, "state change does not match expected")
	require.Equal(t, availabilityResponses, []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}, "availabilityResponses should not have been changed")

	harness.verifyMocks(t)
}

func TestWaitingForChunkBlockCommitFailed(t *testing.T) {
	harness := newBlockSyncHarness()

	event := builders.BlockSyncResponseInput().Build().Message
	availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}

	harness.storage.When("ValidateBlockForCommit", mock.Any).Return(nil, nil).Times(51)
	harness.storage.When("CommitBlock", mock.Any).Call(func(input *services.CommitBlockInput) error {
		if input.BlockPair.ResultsBlock.Header.BlockHeight().Equal(event.SignedChunkRange.FirstBlockHeight() + 50) {
			return errors.New("failed to validate block #51")
		}
		return nil
	}).Times(51)

	newState, availabilityResponses := harness.blockSync.transitionState(BLOCK_SYNC_PETITIONER_WAITING_FOR_CHUNK, event, availabilityResponses, harness.collectAvailabilityTrigger)
	require.Equal(t, BLOCK_SYNC_STATE_IDLE, newState, "state change does not match expected")
	require.Equal(t, availabilityResponses, []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}, "availabilityResponses should not have been changed")

	harness.verifyMocks(t)
}

func TestWaitingForChunkIgnoresInvalidEvents(t *testing.T) {
	events := allEventsExcept(
		"*gossipmessages.BlockSyncResponseMessage",
		"*gossipmessages.BlockAvailabilityRequestMessage",
		"*gossipmessages.BlockSyncRequestMessage")

	for _, event := range events {
		t.Run(typeOfEvent(event), func(t *testing.T) {
			harness := newBlockSyncHarness()

			availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}

			newState, availabilityResponses := harness.blockSync.transitionState(BLOCK_SYNC_PETITIONER_WAITING_FOR_CHUNK, event, availabilityResponses, harness.collectAvailabilityTrigger)

			require.Equal(t, BLOCK_SYNC_PETITIONER_WAITING_FOR_CHUNK, newState, "state change does not match expected")
			require.Equal(t, availabilityResponses, []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}, "availabilityResponses should remain the same")

			harness.verifyMocks(t)
		})
	}
}

func TestAnyStateRespondToAvailabilityRequests(t *testing.T) {
	event := builders.BlockAvailabilityRequestInput().Build().Message

	for _, state := range allStates() {
		t.Run("state="+strconv.Itoa(int(state)), func(t *testing.T) {
			harness := newBlockSyncHarness()

			harness.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(200)).Times(1)
			harness.gossip.When("SendBlockAvailabilityResponse", mock.Any).Return(nil, nil).Times(1)

			availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}

			newState, availabilityResponses := harness.blockSync.transitionState(state, event, availabilityResponses, harness.collectAvailabilityTrigger)

			require.Equal(t, state, newState, "state change was not expected")
			require.Equal(t, availabilityResponses, []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}, "availabilityResponses should remain the same")

			harness.verifyMocks(t)
		})
	}
}

func TestAnyStateRespondsNothingToAvailabilityRequestIfSourceIsBehindPetitioner(t *testing.T) {
	event := builders.BlockAvailabilityRequestInput().Build().Message
	petitionerBlockHeight := event.SignedBatchRange.LastCommittedBlockHeight()

	for _, state := range allStates() {
		t.Run("state="+strconv.Itoa(int(state)), func(t *testing.T) {
			harness := newBlockSyncHarness()

			harness.storage.When("LastCommittedBlockHeight").Return(petitionerBlockHeight).Times(1)
			harness.gossip.Never("SendBlockAvailabilityResponse", mock.Any)

			availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}

			newState, availabilityResponses := harness.blockSync.transitionState(state, event, availabilityResponses, harness.collectAvailabilityTrigger)

			require.Equal(t, state, newState, "state change was not expected")
			require.Equal(t, availabilityResponses, []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}, "availabilityResponses should remain the same")

			harness.verifyMocks(t)
		})
	}
}

func TestAnyStateIgnoresSendBlockAvailabilityRequestsIfFailedToRespond(t *testing.T) {
	event := builders.BlockAvailabilityRequestInput().Build().Message

	for _, state := range allStates() {
		t.Run("state="+strconv.Itoa(int(state)), func(t *testing.T) {
			harness := newBlockSyncHarness()

			harness.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(200)).Times(1)
			harness.gossip.When("SendBlockAvailabilityResponse", mock.Any).Return(nil, errors.New("gossip failure")).Times(1)

			availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}

			newState, availabilityResponses := harness.blockSync.transitionState(state, event, availabilityResponses, harness.collectAvailabilityTrigger)

			require.Equal(t, state, newState, "state change was not expected")
			require.Equal(t, availabilityResponses, []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}, "availabilityResponses should remain the same")

			harness.verifyMocks(t)
		})
	}
}

func TestAnyStateRespondsWithChunks(t *testing.T) {
	event := builders.BlockSyncRequestInput().Build().Message

	firstHeight := primitives.BlockHeight(11)
	lastHeight := primitives.BlockHeight(20)

	var blocks []*protocol.BlockPairContainer

	for i := firstHeight; i <= lastHeight; i++ {
		blocks = append(blocks, builders.BlockPair().WithHeight(i).Build())
	}

	for _, state := range allStates() {
		t.Run("state="+strconv.Itoa(int(state)), func(t *testing.T) {
			harness := newBlockSyncHarness()

			harness.storage.When("GetBlocks").Return(blocks, firstHeight, lastHeight).Times(1)
			harness.storage.When("LastCommittedBlockHeight").Return(lastHeight).Times(1)
			harness.gossip.When("SendBlockSyncResponse", mock.Any).Return(nil, nil).Times(1)

			availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}

			newState, availabilityResponses := harness.blockSync.transitionState(state, event, availabilityResponses, harness.collectAvailabilityTrigger)

			require.Equal(t, state, newState, "state change was not expected")
			require.Equal(t, availabilityResponses, []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}, "availabilityResponses should remain the same")

			harness.verifyMocks(t)
		})
	}
}

func TestEveryStateRestartsSyncAfterHardReset(t *testing.T) {
	for _, state := range allStates() {
		t.Run("state="+strconv.Itoa(int(state)), func(t *testing.T) {
			harness := newBlockSyncHarness()

			availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}

			harness.storage.When("UpdateConsensusAlgosAboutLatestCommittedBlock").Return().Times(1)
			harness.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(10)).Times(1)
			harness.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any).Return(nil, nil).Times(1)
			harness.collectAvailabilityTrigger.When("Reset").Return().Times(1)

			event := hardResetEvent{}

			newState, availabilityResponses := harness.blockSync.transitionState(state, event, availabilityResponses, harness.collectAvailabilityTrigger)

			require.Equal(t, BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES, newState, "state change does not match expected")
			require.Empty(t, availabilityResponses, "no availabilityResponses were sent yet")

			harness.verifyMocks(t)
		})
	}
}

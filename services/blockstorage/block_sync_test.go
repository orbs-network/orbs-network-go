package blockstorage

import (
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

var blockSyncStateNameLookup = map[blockSyncState]string{
	BLOCK_SYNC_STATE_IDLE:                                   `BLOCK_SYNC_STATE_IDLE`,
	BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES: `BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES`,
	BLOCK_SYNC_PETITIONER_WAITING_FOR_CHUNK:                 `BLOCK_SYNC_PETITIONER_WAITING_FOR_CHUNK`,
}

type blockSyncHarness struct {
	blockSync      *BlockSync
	gossip         *gossiptopics.MockBlockSync
	storage        *blockSyncStorageMock
	startSyncTimer *synchronization.PeriodicalTriggerMock
}

func newBlockSyncHarness() *blockSyncHarness {
	cfg := config.ForBlockStorageTests(keys.Ed25519KeyPairForTests(0).PublicKey())
	gossip := &gossiptopics.MockBlockSync{}
	storage := &blockSyncStorageMock{}
	collectAvailabilityTrigger := &synchronization.PeriodicalTriggerMock{}

	blockSync := &BlockSync{
		logger:  log.GetLogger(),
		config:  cfg,
		storage: storage,
		gossip:  gossip,
		events:  nil,
	}

	return &blockSyncHarness{
		blockSync:      blockSync,
		gossip:         gossip,
		storage:        storage,
		startSyncTimer: collectAvailabilityTrigger,
	}
}

func (h *blockSyncHarness) verifyMocks(t *testing.T) {
	ok, err := mock.VerifyMocks(h.storage, h.gossip, h.startSyncTimer)
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

func allStates(collecting bool) []blockSyncState {
	if collecting {
		return []blockSyncState{
			BLOCK_SYNC_STATE_IDLE,
			BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES,
			BLOCK_SYNC_PETITIONER_WAITING_FOR_CHUNK,
		}
	} else {
		return []blockSyncState{
			BLOCK_SYNC_STATE_IDLE,
			BLOCK_SYNC_PETITIONER_WAITING_FOR_CHUNK,
		}
	}
}

func TestSourceAnyStateRespondToAvailabilityRequests(t *testing.T) {
	event := builders.BlockAvailabilityRequestInput().Build().Message

	for _, state := range allStates(true) {
		t.Run("state="+blockSyncStateNameLookup[state], func(t *testing.T) {
			harness := newBlockSyncHarness()

			harness.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(200)).Times(1)
			harness.gossip.When("SendBlockAvailabilityResponse", mock.Any).Return(nil, nil).Times(1)

			availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}

			newState, availabilityResponses := harness.blockSync.transitionState(state, event, availabilityResponses, harness.startSyncTimer)

			require.Equal(t, state, newState, "state change was not expected")
			require.Equal(t, availabilityResponses, []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}, "availabilityResponses should remain the same")

			harness.verifyMocks(t)
		})
	}
}

func TestSourceAnyStateRespondsNothingToAvailabilityRequestIfSourceIsBehindPetitioner(t *testing.T) {
	event := builders.BlockAvailabilityRequestInput().Build().Message
	petitionerBlockHeight := event.SignedBatchRange.LastCommittedBlockHeight()

	for _, state := range allStates(true) {
		t.Run("state="+blockSyncStateNameLookup[state], func(t *testing.T) {
			harness := newBlockSyncHarness()

			harness.storage.When("LastCommittedBlockHeight").Return(petitionerBlockHeight).Times(1)
			harness.gossip.Never("SendBlockAvailabilityResponse", mock.Any)

			availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}

			newState, availabilityResponses := harness.blockSync.transitionState(state, event, availabilityResponses, harness.startSyncTimer)

			require.Equal(t, state, newState, "state change was not expected")
			require.Equal(t, availabilityResponses, []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}, "availabilityResponses should remain the same")

			harness.verifyMocks(t)
		})
	}
}

func TestSourceAnyStateIgnoresSendBlockAvailabilityRequestsIfFailedToRespond(t *testing.T) {
	event := builders.BlockAvailabilityRequestInput().Build().Message

	for _, state := range allStates(true) {
		t.Run("state="+blockSyncStateNameLookup[state], func(t *testing.T) {
			harness := newBlockSyncHarness()

			harness.storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(200)).Times(1)
			harness.gossip.When("SendBlockAvailabilityResponse", mock.Any).Return(nil, errors.New("gossip failure")).Times(1)

			availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}

			newState, availabilityResponses := harness.blockSync.transitionState(state, event, availabilityResponses, harness.startSyncTimer)

			require.Equal(t, state, newState, "state change was not expected")
			require.Equal(t, availabilityResponses, []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}, "availabilityResponses should remain the same")

			harness.verifyMocks(t)
		})
	}
}

func TestSourceAnyStateRespondsWithChunks(t *testing.T) {
	event := builders.BlockSyncRequestInput().Build().Message

	firstHeight := primitives.BlockHeight(11)
	lastHeight := primitives.BlockHeight(20)

	var blocks []*protocol.BlockPairContainer

	for i := firstHeight; i <= lastHeight; i++ {
		blocks = append(blocks, builders.BlockPair().WithHeight(i).Build())
	}

	for _, state := range allStates(true) {
		t.Run("state="+blockSyncStateNameLookup[state], func(t *testing.T) {
			harness := newBlockSyncHarness()

			harness.storage.When("GetBlocks").Return(blocks, firstHeight, lastHeight).Times(1)
			harness.storage.When("LastCommittedBlockHeight").Return(lastHeight).Times(1)
			harness.gossip.When("SendBlockSyncResponse", mock.Any).Return(nil, nil).Times(1)

			availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}

			newState, availabilityResponses := harness.blockSync.transitionState(state, event, availabilityResponses, harness.startSyncTimer)

			require.Equal(t, state, newState, "state change was not expected")
			require.Equal(t, availabilityResponses, []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}, "availabilityResponses should remain the same")

			harness.verifyMocks(t)
		})
	}
}

func TestSourceAnyStateIgnoresBlockSyncRequestIfSourceIsBehindOrInSync(t *testing.T) {
	firstHeight := primitives.BlockHeight(11)
	lastHeight := primitives.BlockHeight(10)

	event := builders.BlockSyncRequestInput().WithFirstBlockHeight(firstHeight).WithLastCommittedBlockHeight(lastHeight).Build().Message

	for _, state := range allStates(true) {
		t.Run("state="+blockSyncStateNameLookup[state], func(t *testing.T) {
			harness := newBlockSyncHarness()

			harness.storage.When("LastCommittedBlockHeight").Return(lastHeight).Times(1)
			harness.storage.Never("GetBlocks")
			harness.gossip.Never("SendBlockSyncResponse", mock.Any)

			availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}

			newState, availabilityResponses := harness.blockSync.transitionState(state, event, availabilityResponses, harness.startSyncTimer)

			require.Equal(t, state, newState, "state change was not expected")
			require.Equal(t, availabilityResponses, []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}, "availabilityResponses should remain the same")

			harness.verifyMocks(t)
		})
	}
}

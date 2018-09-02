package blockstorage

import (
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/stretchr/testify/require"
	"testing"
)

type blockSyncStorageMock struct {
	mock.Mock
}

func (s *blockSyncStorageMock) GetBlocks(first primitives.BlockHeight, last primitives.BlockHeight) (blocks []*protocol.BlockPairContainer, firstAvailableBlockHeight primitives.BlockHeight, lastAvailableBlockHeight primitives.BlockHeight) {
	panic("not mocked")
}

func (s *blockSyncStorageMock) LastCommittedBlockHeight() primitives.BlockHeight {
	ret := s.Called()
	return ret.Get(0).(primitives.BlockHeight)
}

func (s *blockSyncStorageMock) CommitBlock(input *services.CommitBlockInput) (*services.CommitBlockOutput, error) {
	panic("not mocked")
}

func (s *blockSyncStorageMock) ValidateBlockForCommit(input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error) {
	panic("not mocked")
}

func (s *blockSyncStorageMock) UpdateConsensusAlgosAboutLatestCommittedBlock() {
	s.Called()
}

func TestTransitionFromSyncStartToCollectAvailabilityResponses(t *testing.T) {
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

	var event interface{}
	var blockAvailabilityResponses []*gossipmessages.BlockAvailabilityResponseMessage

	storage.When("UpdateConsensusAlgosAboutLatestCommittedBlock").Return().Times(1)
	storage.When("LastCommittedBlockHeight").Return(primitives.BlockHeight(10)).Times(1)

	gossip.When("BroadcastBlockAvailabilityRequest", mock.Any).Return(nil, nil).Times(1)

	newState, responses := blockSync.transitionState(BLOCK_SYNC_STATE_START_SYNC, event, blockAvailabilityResponses)

	require.Equal(t, BLOCK_SYNC_PETITIONER_COLLECTING_AVAILABILITY_RESPONSES, newState)
	require.Empty(t, responses, "no responses were sent yet")

	ok, err := mock.VerifyMocks(storage, gossip)
	require.NoError(t, err)
	require.True(t, ok)
}

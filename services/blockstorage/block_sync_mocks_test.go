package blockstorage

import (
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
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
	ret := s.Called(input)
	return nil, ret.Error(0)
}

func (s *blockSyncStorageMock) ValidateBlockForCommit(input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error) {
	ret := s.Called(input)
	return nil, ret.Error(0)
}

func (s *blockSyncStorageMock) UpdateConsensusAlgosAboutLatestCommittedBlock() {
	s.Called()
}

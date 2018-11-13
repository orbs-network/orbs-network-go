package sync

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type blockSyncStorageMock struct {
	mock.Mock
}

func (s *blockSyncStorageMock) GetLastCommittedBlockHeight(ctx context.Context, input *services.GetLastCommittedBlockHeightInput) (*services.GetLastCommittedBlockHeightOutput, error) {
	ret := s.Called(ctx, input)
	if out := ret.Get(0); out != nil {
		return out.(*services.GetLastCommittedBlockHeightOutput), ret.Error(1)
	} else {
		return nil, ret.Error(1)
	}
}

func (s *blockSyncStorageMock) CommitBlock(ctx context.Context, input *services.CommitBlockInput) (*services.CommitBlockOutput, error) {
	ret := s.Called(ctx, input)
	if out := ret.Get(0); out != nil {
		return out.(*services.CommitBlockOutput), ret.Error(1)
	} else {
		return nil, ret.Error(1)
	}
}

func (s *blockSyncStorageMock) ValidateBlockForCommit(ctx context.Context, input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error) {
	ret := s.Called(ctx, input)
	if out := ret.Get(0); out != nil {
		return out.(*services.ValidateBlockForCommitOutput), ret.Error(1)
	} else {
		return nil, ret.Error(1)
	}
}

func (s *blockSyncStorageMock) UpdateConsensusAlgosAboutLatestCommittedBlock(ctx context.Context) {
	s.Called(ctx)
}

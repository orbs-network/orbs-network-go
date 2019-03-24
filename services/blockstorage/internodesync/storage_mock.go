// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package internodesync

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

func (s *blockSyncStorageMock) NodeSyncCommitBlock(ctx context.Context, input *services.CommitBlockInput) (*services.CommitBlockOutput, error) {
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

func (s *blockSyncStorageMock) UpdateConsensusAlgosAboutLastCommittedBlockInLocalPersistence(ctx context.Context) {
	s.Called(ctx)
}

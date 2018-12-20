package blockstorage

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

// TODO(v1): this function should return an error
func (s *service) UpdateConsensusAlgosAboutLatestCommittedBlock(ctx context.Context) {
	// the source of truth for the last committed block is persistence
	lastCommittedBlock, err := s.persistence.GetLastBlock()
	if err != nil {
		s.logger.Error(err.Error())
		return
	}

	// passing nil on purpose, see spec
	// If lastCommittedBlock is nil, it means this is the Genesis Block
	err = s.validateWithConsensusAlgos(ctx, nil, lastCommittedBlock)
	if err != nil {
		s.logger.Error(err.Error())
		return
	}
}

func (s *service) HandleBlockAvailabilityResponse(ctx context.Context, input *gossiptopics.BlockAvailabilityResponseInput) (*gossiptopics.EmptyOutput, error) {
	if s.nodeSync != nil {
		s.nodeSync.HandleBlockAvailabilityResponse(ctx, input)
	}
	return nil, nil
}

func (s *service) HandleBlockSyncResponse(ctx context.Context, input *gossiptopics.BlockSyncResponseInput) (*gossiptopics.EmptyOutput, error) {
	if s.nodeSync != nil {
		s.nodeSync.HandleBlockSyncResponse(ctx, input)
	}
	return nil, nil
}

package blockstorage

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
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
	// TODO (v1) Do we want to check lastCommittedBlock=nil or not??

	// If lastCommittedBlock is nil, it means this is the Genesis Block
	err = s.notifyConsensusAlgos(ctx, nil, lastCommittedBlock, handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY)
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

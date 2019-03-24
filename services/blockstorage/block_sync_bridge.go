// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package blockstorage

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

// TODO(v1): this function should return an error
func (s *service) UpdateConsensusAlgosAboutLastCommittedBlockInLocalPersistence(ctx context.Context) {
	// the source of truth for the last committed block is persistence
	lastCommittedBlock, err := s.persistence.GetLastBlock()
	if err != nil {
		s.logger.Error("UpdateConsensusAlgosAboutLastCommittedBlockInLocalPersistence(): GetLastBlock() failed", log.Error(err))
		return
	}

	var blockHeight primitives.BlockHeight
	if lastCommittedBlock != nil {
		blockHeight = lastCommittedBlock.TransactionsBlock.Header.BlockHeight()
	}

	s.logger.Info("UpdateConsensusAlgosAboutLatestCommittedBlock calling notifyConsensusAlgos with UPDATE_ONLY", log.BlockHeight(blockHeight))
	err = s.notifyConsensusAlgos(
		ctx,
		nil,                // don't care about prev block, we are updating consensus algo about last committed, not asking it to validate using the prev block
		lastCommittedBlock, // if lastCommittedBlock is nil, it means this is the Genesis Block
		handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY)
	if err != nil {
		s.logger.Error("UpdateConsensusAlgosAboutLastCommittedBlockInLocalPersistence(): notifyConsensusAlgos() failed", log.Error(err))
		return
	} else {
		s.logger.Info("UpdateConsensusAlgosAboutLatestCommittedBlock returned from notifyConsensusAlgos with UPDATE_ONLY", log.BlockHeight(blockHeight))
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

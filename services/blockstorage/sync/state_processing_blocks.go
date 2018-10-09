package sync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type processingBlocksState struct {
	blocks  *gossipmessages.BlockSyncResponseMessage
	logger  log.BasicLogger
	storage BlockSyncStorage
	sf      *stateFactory
}

func (s *processingBlocksState) name() string {
	return "processing-blocks-state"
}

func (s *processingBlocksState) processState(ctx context.Context) syncState {
	firstBlockHeight := s.blocks.SignedChunkRange.FirstBlockHeight()
	lastBlockHeight := s.blocks.SignedChunkRange.LastBlockHeight()

	s.logger.Info("committing blocks from sync",
		log.Stringable("sender", s.blocks.Sender),
		log.Stringable("first-block-height", firstBlockHeight),
		log.Stringable("last-block-height", lastBlockHeight))

	for _, blockPair := range s.blocks.BlockPairs {
		_, err := s.storage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{BlockPair: blockPair})

		if err != nil {
			s.logger.Error("failed to validate block received via sync", log.Error(err))
			break
		}

		_, err = s.storage.CommitBlock(&services.CommitBlockInput{BlockPair: blockPair})

		if err != nil {
			s.logger.Error("failed to commit block received via sync", log.Error(err))
			break
		}
	}

	return s.sf.CreateCollectingAvailabilityResponseState()
}

func (s *processingBlocksState) blockCommitted(blockHeight primitives.BlockHeight) {
	panic("implement me")
}

func (s *processingBlocksState) gotAvailabilityResponse(message *gossipmessages.BlockAvailabilityResponseMessage) {
	panic("implement me")
}

func (s *processingBlocksState) gotBlocks(message *gossipmessages.BlockSyncResponseMessage) {
	panic("implement me")
}

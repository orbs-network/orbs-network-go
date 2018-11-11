package sync

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
)

type processingBlocksState struct {
	blocks  *gossipmessages.BlockSyncResponseMessage
	logger  log.BasicLogger
	storage BlockSyncStorage
	factory *stateFactory
	metrics processingStateMetrics
}

func (s *processingBlocksState) name() string {
	return "processing-blocks-state"
}

func (s *processingBlocksState) String() string {
	if s.blocks != nil {
		return fmt.Sprintf("%s-with-%d-blocks", s.name(), len(s.blocks.BlockPairs))
	}

	return s.name()
}

func (s *processingBlocksState) processState(ctx context.Context) syncState {
	start := time.Now()
	defer s.metrics.stateLatency.RecordSince(start) // runtime metric

	if ctx.Err() == context.Canceled { // system is terminating and we do not select on channels in this state
		return nil
	}

	if s.blocks == nil {
		s.logger.Info("possible byzantine state in block sync, received no blocks to processing blocks state")
		return s.factory.CreateIdleState()
	}

	firstBlockHeight := s.blocks.SignedChunkRange.FirstBlockHeight()
	lastBlockHeight := s.blocks.SignedChunkRange.LastBlockHeight()

	s.logger.Info("committing blocks from sync",
		log.Int("block-count", len(s.blocks.BlockPairs)),
		log.Stringable("sender", s.blocks.Sender),
		log.Stringable("first-block-height", firstBlockHeight),
		log.Stringable("last-block-height", lastBlockHeight))

	for _, blockPair := range s.blocks.BlockPairs {
		s.metrics.blocksRate.Measure(1)
		_, err := s.storage.ValidateBlockForCommit(ctx, &services.ValidateBlockForCommitInput{BlockPair: blockPair})

		if err != nil {
			s.metrics.failedValidationBlocks.Inc()
			s.logger.Error("failed to validate block received via sync", log.Error(err), log.BlockHeight(blockPair.TransactionsBlock.Header.BlockHeight()), log.Stringable("tx-block", blockPair.TransactionsBlock))
			break
		}

		_, err = s.storage.CommitBlock(ctx, &services.CommitBlockInput{BlockPair: blockPair})

		if err != nil {
			s.metrics.failedCommitBlocks.Inc()
			s.logger.Error("failed to commit block received via sync", log.Error(err), log.BlockHeight(blockPair.TransactionsBlock.Header.BlockHeight()))
			break
		} else {
			s.metrics.committedBlocks.Inc()
			s.logger.Info("successfully committed block received via sync", log.BlockHeight(blockPair.TransactionsBlock.Header.BlockHeight()))
		}
	}

	return s.factory.CreateCollectingAvailabilityResponseState()
}

func (s *processingBlocksState) blockCommitted(ctx context.Context) {
	return
}

func (s *processingBlocksState) gotAvailabilityResponse(ctx context.Context, message *gossipmessages.BlockAvailabilityResponseMessage) {
	return
}

func (s *processingBlocksState) gotBlocks(ctx context.Context, message *gossipmessages.BlockSyncResponseMessage) {
	return
}

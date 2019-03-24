// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package internodesync

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
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
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	start := time.Now()
	defer s.metrics.timeSpentInState.RecordSince(start) // runtime metric

	if s.blocks == nil {
		s.logger.Info("possible byzantine state in block sync, received no blocks to processing blocks state")
		return s.factory.CreateIdleState()
	}

	firstBlockHeight := s.blocks.SignedChunkRange.FirstBlockHeight()
	lastBlockHeight := s.blocks.SignedChunkRange.LastBlockHeight()

	numBlocks := len(s.blocks.BlockPairs)
	logger.Info("committing blocks from sync",
		log.Int("block-count", numBlocks),
		log.Stringable("sender", s.blocks.Sender),
		log.Uint64("first-block-height", uint64(firstBlockHeight)),
		log.Uint64("last-block-height", uint64(lastBlockHeight)))

	s.metrics.blocksRate.Measure(int64(numBlocks))
	for _, blockPair := range s.blocks.BlockPairs {
		if !s.factory.conduit.drainAndCheckForShutdown(ctx) {
			return nil
		}
		_, err := s.storage.ValidateBlockForCommit(ctx, &services.ValidateBlockForCommitInput{BlockPair: blockPair})

		if err != nil {
			s.metrics.failedValidationBlocks.Inc()
			logger.Info("failed to validate block received via sync", log.Error(err), log.BlockHeight(blockPair.TransactionsBlock.Header.BlockHeight()), log.Stringable("tx-block", blockPair.TransactionsBlock)) // may be a valid failure if height isn't the next height
			break
		}

		_, err = s.storage.NodeSyncCommitBlock(ctx, &services.CommitBlockInput{BlockPair: blockPair})

		if err != nil {
			s.metrics.failedCommitBlocks.Inc()
			logger.Error("failed to commit block received via sync", log.Error(err), log.BlockHeight(blockPair.TransactionsBlock.Header.BlockHeight()))
			break
		} else {
			s.metrics.lastCommittedTime.Update(time.Now().UnixNano())
			s.metrics.committedBlocks.Inc()
			logger.Info("successfully committed block received via sync", log.BlockHeight(blockPair.TransactionsBlock.Header.BlockHeight()))
		}
	}

	if !s.factory.conduit.drainAndCheckForShutdown(ctx) {
		return nil
	}

	return s.factory.CreateCollectingAvailabilityResponseState()
}

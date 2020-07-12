// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package internodesync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/orbs-network/crypto-lib-go/crypto/digest"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
	"time"
)

type processingBlocksState struct {
	blocks  *gossipmessages.BlockSyncResponseMessage
	logger  log.Logger
	storage BlockSyncStorage
	factory *stateFactory
	conduit blockSyncConduit
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

	if s.blocks == nil || len(s.blocks.BlockPairs) == 0 {
		s.logger.Info("possible byzantine state in block sync, received no blocks to processing blocks state")
		return s.factory.CreateIdleState()
	}

	firstBlockHeight := s.blocks.SignedChunkRange.FirstBlockHeight()
	lastBlockHeight := s.blocks.SignedChunkRange.LastBlockHeight()

	numBlocks := len(s.blocks.BlockPairs)
	logger.Info("processing blocks from sync",
		log.Int("block-count", numBlocks),
		log.Stringable("sender", s.blocks.Sender),
		log.Uint64("first-block-height", uint64(firstBlockHeight)),
		log.Uint64("last-block-height", uint64(lastBlockHeight)))

	receivedSyncBlocksOrder := s.blocks.SignedChunkRange.BlocksOrder()
	syncState := s.storage.GetSyncState()
	if err := s.validateBlocksRange(s.blocks.BlockPairs, syncState, receivedSyncBlocksOrder); err != nil {
		s.metrics.failedValidationBlocks.Inc()
		logger.Info("failed to verify the blocks chunk range received via sync", log.Error(err))
		return s.factory.CreateCollectingAvailabilityResponseState()
	}
	if err := s.validatePosChain(s.blocks.BlockPairs, syncState, s.factory.config.BlockSyncReferenceMaxAllowedDistance(), receivedSyncBlocksOrder); err != nil {
		s.metrics.failedValidationBlocks.Inc()
		logger.Info("failed to verify the blocks chunk PoS received via sync", log.Error(err))
		return s.factory.CreateCollectingAvailabilityResponseState()
	}

	s.metrics.blocksRate.Measure(int64(numBlocks))

	for index, blockPair := range s.blocks.BlockPairs {
		if !s.conduit.drainAndCheckForShutdown(ctx) {
			return nil
		}
		prevBlockPair := s.getPrevBlock(index, receivedSyncBlocksOrder)
		_, err := s.storage.ValidateBlockForCommit(ctx, &services.ValidateBlockForCommitInput{BlockPair: blockPair, PrevBlockPair: prevBlockPair})
		if err != nil {
			s.metrics.failedValidationBlocks.Inc()
			logger.Info("failed to validate block received via sync", log.Error(err), logfields.BlockHeight(blockPair.TransactionsBlock.Header.BlockHeight()), log.Stringable("tx-block-header", blockPair.TransactionsBlock.Header)) // may be a valid failure if height isn't the next height
			break
		}
		_, err = s.storage.NodeSyncCommitBlock(ctx, &services.CommitBlockInput{BlockPair: blockPair})
		if err != nil {
			s.metrics.failedCommitBlocks.Inc()
			logger.Error("failed to commit block received via sync", log.Error(err), logfields.BlockHeight(blockPair.TransactionsBlock.Header.BlockHeight()))
			break
		} else {
			s.metrics.lastCommittedTime.Update(time.Now().UnixNano())
			s.metrics.committedBlocks.Inc()
			logger.Info("successfully committed block received via sync", logfields.BlockHeight(blockPair.TransactionsBlock.Header.BlockHeight()))
		}
	}

	govnr.Once(logfields.GovnrErrorer(logger), func() {
		shortCtx, cancel := context.WithTimeout(ctx, time.Second) // TODO V1 move timeout to configuration
		defer cancel()
		s.storage.UpdateConsensusAlgosAboutLastCommittedBlockInLocalPersistence(shortCtx)
	})

	if !s.conduit.drainAndCheckForShutdown(ctx) {
		return nil
	}

	return s.factory.CreateCollectingAvailabilityResponseState()
}

func (s *processingBlocksState) validateBlocksRange(blocks []*protocol.BlockPairContainer, syncState SyncState, receivedSyncBlocksOrder gossipmessages.SyncBlocksOrder) error {
	syncBlocksOrder := s.factory.getSyncBlocksOrder()
	if receivedSyncBlocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_RESERVED && syncBlocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_ASCENDING {
		prevHeight := syncState.InOrderHeight
		for _, blockPair := range s.blocks.BlockPairs {
			currentHeight := blockPair.TransactionsBlock.Header.BlockHeight()
			if currentHeight != prevHeight+1 {
				return fmt.Errorf("invalid blocks chunk found a non consecutive ascending range prevHeight (%d), currentHeight (%d)", prevHeight, currentHeight)
			}
			prevHeight = currentHeight
		}
		return nil

	} else if syncBlocksOrder != receivedSyncBlocksOrder {
		return errors.New("received chunk with blocks order which does not match blockSync expected blocks order")

	} else if receivedSyncBlocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_DESCENDING {
		firstBlock := blocks[0]
		firstBlockHeight := firstBlock.TransactionsBlock.Header.BlockHeight()
		if syncState.InOrderHeight == syncState.TopHeight {
			if firstBlockHeight <= syncState.InOrderHeight {
				return fmt.Errorf("invalid blocks chunk where first block height (%d) < syncState.inOrderHeight (%d)", firstBlockHeight, syncState.InOrderHeight)
			}
		} else if syncState.InOrderHeight < syncState.TopHeight { // blocks chunk should range from lastSynced-1 down
			if firstBlockHeight != syncState.LastSyncedHeight-1 {
				return fmt.Errorf("invalid blocks chunk where first block height (%d) != syncState.LastSyncedHeight(%d) -1, inorder(%d), top(%d) ", firstBlockHeight, syncState.LastSyncedHeight, syncState.InOrderHeight, syncState.TopHeight)
			}
		}
		prevHeight := firstBlockHeight + 1
		for _, blockPair := range s.blocks.BlockPairs {
			currentHeight := blockPair.TransactionsBlock.Header.BlockHeight()
			if currentHeight+1 != prevHeight {
				return fmt.Errorf("invalid blocks chunk found a non consecutive descending range prevHeight (%d), currentHeight (%d)", prevHeight, currentHeight)
			}
			prevHeight = currentHeight
		}
		return nil
	}
	return nil
}

// assumes blocks range is correct. Specifically in descending (blockStorage.lastSynced.height - 1 == blocks[0].height ) or ( blocks[0].height > blockStorage.top.height)
func (s *processingBlocksState) validatePosChain(blocks []*protocol.BlockPairContainer, syncState SyncState, committeeValidityGraceTimeout time.Duration, receivedSyncBlocksOrder gossipmessages.SyncBlocksOrder) error {
	syncBlocksOrder := s.factory.getSyncBlocksOrder()
	if receivedSyncBlocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_RESERVED && syncBlocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_ASCENDING {
		return nil

	} else if receivedSyncBlocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_DESCENDING {
		firstBlock := blocks[0]
		firstBlockHeight := firstBlock.TransactionsBlock.Header.BlockHeight()
		if firstBlockHeight == syncState.LastSyncedHeight-1 { // will verify hash pointer to block
			if nextBlock, err := s.storage.GetBlock(firstBlock.TransactionsBlock.Header.BlockHeight() + 1); err == nil && nextBlock != nil {
				// prepend
				blocks = append([]*protocol.BlockPairContainer{nextBlock}, blocks...)
			} else {
				return err
			}
		} else if firstBlockHeight > syncState.TopHeight { // verify the first block reference complies with committee PoS honesty assumption
			topBlockReference := firstBlock.TransactionsBlock.Header.ReferenceTime()
			now := primitives.TimestampSeconds(time.Now().Unix())
			if topBlockReference+primitives.TimestampSeconds(committeeValidityGraceTimeout/time.Second) < now {
				return errors.New(fmt.Sprintf("block reference is not included in committee valid reference grace:  block reference (%d), now (%d), grace (%d)", topBlockReference, now, primitives.TimestampSeconds(committeeValidityGraceTimeout/time.Second)))
			}
		} else {
			return errors.New(fmt.Sprintf("blocks chunk received (firstHeight %d) does not match current syncState (%v)", firstBlockHeight, syncState))
		}

		for i := 0; i < len(blocks)-1; i++ {
			blockPair := blocks[i]
			prevBlockPair := blocks[i+1]
			if !verifyPrevHashPointer(blockPair, prevBlockPair) {
				return errors.New(fmt.Sprintf("prevBlockHash mismatch: block %v; prevBlock: %v", blockPair.String(), prevBlockPair.String()))
			}
		}
	}
	return nil
}

func verifyPrevHashPointer(blockPair *protocol.BlockPairContainer, prevBlockPair *protocol.BlockPairContainer) bool {
	if !bytes.Equal(blockPair.TransactionsBlock.Header.PrevBlockHashPtr(), digest.CalcTransactionsBlockHash(prevBlockPair.TransactionsBlock)) {
		return false
	}

	if !bytes.Equal(blockPair.ResultsBlock.Header.PrevBlockHashPtr(), digest.CalcResultsBlockHash(prevBlockPair.ResultsBlock)) {
		return false
	}
	return true
}

func (s *processingBlocksState) getPrevBlock(index int, receivedSyncBlocksOrder gossipmessages.SyncBlocksOrder) (prevBlock *protocol.BlockPairContainer) {
	blocks := s.blocks.BlockPairs
	blockHeight := blocks[index].TransactionsBlock.Header.BlockHeight()
	if receivedSyncBlocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_ASCENDING {
		if index == 0 {
			if blockHeight > 0 {
				prevBlock, _ = s.storage.GetBlock(blockHeight - 1)
			}
		} else {
			prevBlock = blocks[index-1]
		}
	} else if receivedSyncBlocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_DESCENDING {
		if index == len(blocks)-1 {
			if blockHeight > 0 {
				prevBlock, _ = s.storage.GetBlock(blockHeight - 1)
			}
		} else {
			prevBlock = blocks[index+1]
		}
	}
	return
}

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
	"github.com/orbs-network/orbs-network-go/config"
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
	blocks                   *gossipmessages.BlockSyncResponseMessage
	logger                   log.Logger
	storage                  BlockSyncStorage
	factory                  *stateFactory
	conduit                  blockSyncConduit
	syncBlocksOrder          gossipmessages.SyncBlocksOrder
	tempSyncStorage          TempSyncStorage
	management               services.Management
	referenceMaxDistance     time.Duration
	managementReferenceGrace time.Duration
	metrics                  processingStateMetrics
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
		s.logger.Info("possible byzantine state in Block sync, received no blocks to processing blocks state")
		return s.factory.CreateIdleState()
	}

	firstBlockHeight := s.blocks.SignedChunkRange.FirstBlockHeight()
	lastBlockHeight := s.blocks.SignedChunkRange.LastBlockHeight()

	numBlocks := len(s.blocks.BlockPairs)
	logger.Info("committing blocks from sync",
		log.Int("Block-count", numBlocks),
		log.Stringable("sender", s.blocks.Sender),
		log.Uint64("first-Block-height", uint64(firstBlockHeight)),
		log.Uint64("last-Block-height", uint64(lastBlockHeight)))

	receivedSyncBlocksOrder := s.blocks.SignedChunkRange.BlocksOrder()
	err := s.validatePosChain(ctx, s.blocks.BlockPairs, receivedSyncBlocksOrder)
	if err != nil {
		logger.Error("failed to verify the blocks chunk received via sync", log.Error(err))
		return s.factory.CreateIdleState()
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
			logger.Info("failed to validate Block received via sync", log.Error(err), logfields.BlockHeight(blockPair.TransactionsBlock.Header.BlockHeight()), log.Stringable("tx-Block", blockPair.TransactionsBlock)) // may be a valid failure if height isn't the next height
			break
		}

		// TODO: Gad temporary code for PR
		err = commitBlockTemp(ctx, s.tempSyncStorage, blockPair, s.storage, s.logger, s.metrics)
	}

	if !s.conduit.drainAndCheckForShutdown(ctx) {
		return nil
	}

	return s.factory.CreateCollectingAvailabilityResponseState()
}

func (s *processingBlocksState) validatePosChain(ctx context.Context, blocks []*protocol.BlockPairContainer, receivedSyncBlocksOrder gossipmessages.SyncBlocksOrder) error {
	if receivedSyncBlocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_RESERVED && s.syncBlocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_ASCENDING {
		return nil
	} else if s.syncBlocksOrder != receivedSyncBlocksOrder {
		return errors.New("received chunk with blocks order which does not match blockSync")
	}
	if receivedSyncBlocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_DESCENDING {
		//return nil
		// TODO: revert
		firstBlock := blocks[0]
		if nextBlock := s.tempSyncStorage.getBlock(firstBlock.TransactionsBlock.Header.BlockHeight() + 1); nextBlock != nil { // will verify hash pointer to block
			// prepend
			blocks = append([]*protocol.BlockPairContainer{nextBlock}, blocks...)
		} else {
			ref, err := s.management.GetCurrentReference(ctx, &services.GetCurrentReferenceInput{})
			if err != nil {
				s.logger.Error("management.GetCurrentReference should not return error", log.Error(err))
				return err
			}
			currentTime := primitives.TimestampSeconds(time.Now().Unix())
			managementGrace := primitives.TimestampSeconds(s.managementReferenceGrace / time.Second)
			if ref.CurrentReference + managementGrace < currentTime {
				return errors.New(fmt.Sprintf("management.GetCurrentReference(%d) is outdated compared to current time (%d) and allowed grace (%d)", ref.CurrentReference, currentTime, managementGrace))
			}
			if firstBlock.TransactionsBlock.Header.ReferenceTime() + primitives.TimestampSeconds(s.referenceMaxDistance/time.Second) < ref.CurrentReference {
				return errors.New(fmt.Sprintf("Block time reference %d is too far back compared to validator current time reference %d", firstBlock.TransactionsBlock.Header.ReferenceTime(), ref.CurrentReference))
			}
		}

		for i := 0; i < len(blocks)-1; i++ {
			blockPair := blocks[i]
			prevBlockPair := blocks[i+1]
			if !verifyPrevHashPointer(blockPair, prevBlockPair) {
				return errors.New(fmt.Sprintf("Block prevBlockHash mismatches prevBlock: Block %v; prevBlock %v", blockPair.String(), prevBlockPair.String()))
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

func (s *processingBlocksState) getPrevBlock(index int, receivedSyncBlocksOrder gossipmessages.SyncBlocksOrder) *protocol.BlockPairContainer {
	blocks := s.blocks.BlockPairs
	if receivedSyncBlocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_ASCENDING {
		if index == 0 {
			lastBlock, _ := s.storage.GetLastCommittedBlock()
			return lastBlock
		}
		return blocks[index-1]
	}
	if receivedSyncBlocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_DESCENDING {
		if index == len(blocks)-1 {
			return s.tempSyncStorage.getBlock(blocks[index].TransactionsBlock.Header.BlockHeight() - 1)
		}
		return blocks[index+1]
	}
	return nil
}

func commitBlockTemp(ctx context.Context, tempSyncStorage TempSyncStorage, block *protocol.BlockPairContainer, persistentStorage BlockSyncStorage, logger log.Logger, metrics processingStateMetrics) error {
	tempSyncStorage.mutex.Lock()
	defer tempSyncStorage.mutex.Unlock()

	commitBlockHeight := block.TransactionsBlock.Header.BlockHeight()
	logger.Info("Trying to commit a Block to TempSyncStorage", logfields.BlockHeight(commitBlockHeight))
	if block.TransactionsBlock.Header.ProtocolVersion() > config.MAXIMAL_PROTOCOL_VERSION_SUPPORTED_VALUE {
		return fmt.Errorf("protocol version (%d) higher than maximal supported (%d) in transactions Block header", block.TransactionsBlock.Header.ProtocolVersion(), config.MAXIMAL_PROTOCOL_VERSION_SUPPORTED_VALUE)
	}
	if block.ResultsBlock.Header.ProtocolVersion() > config.MAXIMAL_PROTOCOL_VERSION_SUPPORTED_VALUE {
		return fmt.Errorf("protocol version (%d) higher than maximal supported (%d) in results Block header", block.ResultsBlock.Header.ProtocolVersion(), config.MAXIMAL_PROTOCOL_VERSION_SUPPORTED_VALUE)
	}

	if tempSyncStorage.getBlock(commitBlockHeight) == nil {
		tempSyncStorage.blocksMap[commitBlockHeight] = block
	}

	syncState := tempSyncStorage.syncState
	if commitBlockHeight > syncState.Top.Height {
		syncState.Top.Set(block)
	}
	if commitBlockHeight == syncState.TopInOrder.Height+1 {
		var tempBlock *protocol.BlockPairContainer
		//commit to persistent storage
		for height := syncState.TopInOrder.Height + 1; height <= syncState.Top.Height; height++ {
			tempBlock = tempSyncStorage.getBlock(height)
			if tempBlock != nil {
				_, err := persistentStorage.NodeSyncCommitBlock(ctx, &services.CommitBlockInput{BlockPair: tempBlock})
				if err != nil {
					metrics.failedCommitBlocks.Inc()
					logger.Error("failed to commit to persistent storage from temp storage", log.Error(err), logfields.BlockHeight(tempBlock.TransactionsBlock.Header.BlockHeight()))
				} else {
					syncState.TopInOrder.Set(tempBlock)
					metrics.lastCommittedTime.Update(time.Now().UnixNano())
					metrics.committedBlocks.Inc()
					logger.Info("successfully committed Block received via sync", logfields.BlockHeight(tempBlock.TransactionsBlock.Header.BlockHeight()))
				}
			}
		}
		syncState.LastSynced.Set(syncState.Top.Block)
	} else {
		syncState.LastSynced.Set(block)
	}

	return nil
}

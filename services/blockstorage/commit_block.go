// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package blockstorage

import (
	"context"
	"fmt"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"time"
)

func (s *Service) NodeSyncCommitBlock(ctx context.Context, input *services.CommitBlockInput) (*services.CommitBlockOutput, error) {
	return s.commitBlock(ctx, input, false)
}

func (s *Service) CommitBlock(ctx context.Context, input *services.CommitBlockInput) (*services.CommitBlockOutput, error) {
	return s.commitBlock(ctx, input, true)
}

func (s *Service) commitBlock(ctx context.Context, input *services.CommitBlockInput, notifyNodeSync bool) (*services.CommitBlockOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	proposedBlockHeight := input.BlockPair.TransactionsBlock.Header.BlockHeight()
	logger.Info("Trying to commit a block", logfields.BlockHeight(proposedBlockHeight))

	if err := s.validateProtocolVersion(input.BlockPair); err != nil {
		return nil, err
	}

	added, persistedHeight, err := s.persistence.WriteNextBlock(input.BlockPair)
	if err != nil {
		return nil, err
	}

	if proposedBlockHeight > persistedHeight+1 {
		return nil, fmt.Errorf("attempt to write future block %d. current top height is %d", proposedBlockHeight, persistedHeight)
	}

	if !added {
		storedRsBlock, err := s.persistence.GetResultsBlock(proposedBlockHeight)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load results block at proposed block height %d", proposedBlockHeight)
		}

		storedTxBlock, err := s.persistence.GetTransactionsBlock(proposedBlockHeight)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load transactions block at proposed block height %d", proposedBlockHeight)
		}
		return nil, detectForks(input.BlockPair, storedTxBlock.Header, storedRsBlock.Header, logger)
	}

	s.metrics.blockHeight.Update(int64(input.BlockPair.TransactionsBlock.Header.BlockHeight()))
	s.metrics.lastCommittedTime.Update(int64(input.BlockPair.TransactionsBlock.Header.Timestamp()))

	if notifyNodeSync {
		govnr.Once(logfields.GovnrErrorer(logger), func() {
			shortCtx, cancel := context.WithTimeout(ctx, time.Second) // TODO V1 move timeout to configuration
			defer cancel()
			s.nodeSync.HandleBlockCommitted(shortCtx)
		})
	}

	logger.Info("committed a block", logfields.BlockHeight(proposedBlockHeight), log.Int("num-transactions", len(input.BlockPair.TransactionsBlock.SignedTransactions)))

	return nil, nil
}

func detectForks(proposedBlock *protocol.BlockPairContainer, storedTxBlockHeader *protocol.TransactionsBlockHeader, storedRsBlockHeader *protocol.ResultsBlockHeader, logger log.Logger) error {
	txBlockHeader := proposedBlock.TransactionsBlock.Header
	rsBlockHeader := proposedBlock.ResultsBlock.Header
	proposedBlockHeight := txBlockHeader.BlockHeight()

	if txBlockHeader.Timestamp() != storedTxBlockHeader.Timestamp() {
		errorMessage := "FORK!! block already in storage, timestamp mismatch"
		// fork found! this is a major error we must report to logs
		logger.Error(errorMessage, logfields.BlockHeight(proposedBlockHeight), log.Stringable("new-block", txBlockHeader), log.Stringable("existing-block", storedTxBlockHeader))
		return errors.New(errorMessage)
	} else if !txBlockHeader.Equal(storedTxBlockHeader) {
		errorMessage := "FORK!! block already in storage, transaction block header mismatch"
		// fork found! this is a major error we must report to logs
		logger.Error(errorMessage, logfields.BlockHeight(proposedBlockHeight), log.Stringable("new-block", txBlockHeader), log.Stringable("existing-block", storedTxBlockHeader))
		return errors.New(errorMessage)
	} else if !rsBlockHeader.Equal(storedRsBlockHeader) {
		errorMessage := "FORK!! block already in storage, results block header mismatch"
		// fork found! this is a major error we must report to logs
		logger.Error(errorMessage, logfields.BlockHeight(proposedBlockHeight), log.Stringable("new-block", rsBlockHeader), log.Stringable("existing-block", storedRsBlockHeader))
		return errors.New(errorMessage)
	}

	logger.Info("block already in storage, skipping", logfields.BlockHeight(proposedBlockHeight))
	return nil
}

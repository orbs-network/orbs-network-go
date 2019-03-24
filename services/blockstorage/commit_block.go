// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package blockstorage

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
)

func (s *service) NodeSyncCommitBlock(ctx context.Context, input *services.CommitBlockInput) (*services.CommitBlockOutput, error) {
	return s.commitBlock(ctx, input, false)
}

func (s *service) CommitBlock(ctx context.Context, input *services.CommitBlockInput) (*services.CommitBlockOutput, error) {
	return s.commitBlock(ctx, input, true)
}

func (s *service) commitBlock(ctx context.Context, input *services.CommitBlockInput, notifyNodeSync bool) (*services.CommitBlockOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	txBlockHeader := input.BlockPair.TransactionsBlock.Header
	rsBlockHeader := input.BlockPair.ResultsBlock.Header

	logger.Info("Trying to commit a block", log.BlockHeight(txBlockHeader.BlockHeight()))

	if err := s.validateProtocolVersion(input.BlockPair); err != nil {
		return nil, err
	}

	// the source of truth for the last committed block is persistence
	lastCommittedBlock, err := s.persistence.GetLastBlock()
	if err != nil {
		return nil, err
	}

	// TODO(https://github.com/orbs-network/orbs-network-go/issues/524): the logic here aborting commits for already committed blocks is duplicated in the adapter because this is not under lock. synchronize to avoid duplicating logic in adapter
	if ok, err := s.validateBlockDoesNotExist(ctx, txBlockHeader, rsBlockHeader, lastCommittedBlock); err != nil || !ok {
		return nil, err
	}

	if err := s.validateConsecutiveBlockHeight(input.BlockPair, lastCommittedBlock); err != nil {
		return nil, err
	}

	if _, err := s.persistence.WriteNextBlock(input.BlockPair); err != nil {
		return nil, err
	}

	s.metrics.blockHeight.Update(int64(input.BlockPair.TransactionsBlock.Header.BlockHeight()))

	if notifyNodeSync {
		supervised.GoOnce(logger, func() {
			shortCtx, cancel := context.WithTimeout(ctx, time.Second) // TODO V1 move timeout to configuration
			defer cancel()
			s.nodeSync.HandleBlockCommitted(shortCtx)
		})
	}

	logger.Info("committed a block", log.BlockHeight(txBlockHeader.BlockHeight()), log.Int("num-transactions", len(input.BlockPair.TransactionsBlock.SignedTransactions)))

	return nil, nil
}

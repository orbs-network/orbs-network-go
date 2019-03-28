// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package blockstorage

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
)

// TODO Implement optimization for full structural validation here (https://github.com/orbs-network/orbs-network-go/issues/684)
func (s *service) ValidateBlockForCommit(ctx context.Context, input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	if protocolVersionError := s.validateProtocolVersion(input.BlockPair); protocolVersionError != nil {
		return nil, protocolVersionError
	}

	// the source of truth for the last committed block is persistence
	lastCommittedBlock, err := s.persistence.GetLastBlock()
	if err != nil {
		return nil, err
	}

	if blockHeightError := s.validateConsecutiveBlockHeight(input.BlockPair, lastCommittedBlock); blockHeightError != nil {
		return nil, blockHeightError
	}

	logger.Info("ValidateBlockForCommit calling notifyConsensusAlgos with VERIFY_AND_UPDATE", log.BlockHeight(input.BlockPair.TransactionsBlock.Header.BlockHeight()))
	if err := s.notifyConsensusAlgos(ctx, lastCommittedBlock, input.BlockPair, handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE); err != nil {
		if ctx.Err() == nil { // this may fail rightfully on graceful shutdown (ctx.Done), we don't want to report an error in this case
			logger.Error("ValidateBlockForCommit(): notifyConsensusAlgos() failed (block validation by consensus algo failed)", log.Error(err), log.Stringable("tx-block-header", input.BlockPair.TransactionsBlock.Header))
		}
		return nil, err
	} else {
		logger.Info("ValidateBlockForCommit returned from notifyConsensusAlgos with VERIFY_AND_UPDATE", log.BlockHeight(input.BlockPair.TransactionsBlock.Header.BlockHeight()))
	}

	return &services.ValidateBlockForCommitOutput{}, nil
}

// how to check if a block already exists: https://github.com/orbs-network/orbs-spec/issues/50
func (s *service) validateBlockDoesNotExist(ctx context.Context, txBlockHeader *protocol.TransactionsBlockHeader, rsBlockHeader *protocol.ResultsBlockHeader, lastCommittedBlock *protocol.BlockPairContainer) (bool, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	currentBlockHeight := getBlockHeight(lastCommittedBlock)
	attemptedBlockHeight := txBlockHeader.BlockHeight()

	if attemptedBlockHeight < currentBlockHeight {
		// we can't check for fork because we don't have the tx header of the old block easily accessible
		errorMessage := "block already in storage, skipping"
		logger.Info(errorMessage, log.BlockHeight(currentBlockHeight), log.Stringable("attempted-block-height", attemptedBlockHeight))
		return false, nil
	} else if attemptedBlockHeight == currentBlockHeight {
		// we can check for fork because we do have the tx header of the old block easily accessible
		if txBlockHeader.Timestamp() != getBlockTimestamp(lastCommittedBlock) {
			errorMessage := "FORK!! block already in storage, timestamp mismatch"
			// fork found! this is a major error we must report to logs
			logger.Error(errorMessage, log.BlockHeight(currentBlockHeight), log.Stringable("attempted-block-height", attemptedBlockHeight), log.Stringable("new-block", txBlockHeader), log.Stringable("existing-block", lastCommittedBlock.TransactionsBlock.Header))
			return false, errors.New(errorMessage)
		} else if !txBlockHeader.Equal(lastCommittedBlock.TransactionsBlock.Header) {
			errorMessage := "FORK!! block already in storage, transaction block header mismatch"
			// fork found! this is a major error we must report to logs
			logger.Error(errorMessage, log.BlockHeight(currentBlockHeight), log.Stringable("attempted-block-height", attemptedBlockHeight), log.Stringable("new-block", txBlockHeader), log.Stringable("existing-block", lastCommittedBlock.TransactionsBlock.Header))
			return false, errors.New(errorMessage)
		} else if !rsBlockHeader.Equal(lastCommittedBlock.ResultsBlock.Header) {
			errorMessage := "FORK!! block already in storage, results block header mismatch"
			// fork found! this is a major error we must report to logs
			s.logger.Error(errorMessage, log.BlockHeight(currentBlockHeight), log.Stringable("attempted-block-height", attemptedBlockHeight), log.Stringable("new-block", rsBlockHeader), log.Stringable("existing-block", lastCommittedBlock.ResultsBlock.Header))
			return false, errors.New(errorMessage)
		}

		logger.Info("block already in storage, skipping", log.BlockHeight(currentBlockHeight), log.Stringable("attempted-block-height", attemptedBlockHeight))
		return false, nil
	}

	return true, nil
}

func (s *service) validateConsecutiveBlockHeight(blockPair *protocol.BlockPairContainer, lastCommittedBlock *protocol.BlockPairContainer) error {
	expectedBlockHeight := getBlockHeight(lastCommittedBlock) + 1

	txBlockHeader := blockPair.TransactionsBlock.Header
	rsBlockHeader := blockPair.ResultsBlock.Header

	if txBlockHeader.BlockHeight() != rsBlockHeader.BlockHeight() {
		return fmt.Errorf("block pair height mismatch. transactions height is %d, results height is %d", txBlockHeader.BlockHeight(), rsBlockHeader.BlockHeight())
	}

	if txBlockHeader.BlockHeight() != expectedBlockHeight {
		return fmt.Errorf("block height is %d, expected %d", txBlockHeader.BlockHeight(), expectedBlockHeight)
	}

	return nil
}

func (s *service) validateProtocolVersion(blockPair *protocol.BlockPairContainer) error {
	txBlockHeader := blockPair.TransactionsBlock.Header
	rsBlockHeader := blockPair.ResultsBlock.Header

	// TODO(v1) we may be logging twice, this should be fixed when handling the logging structured errors in logger issue
	if !txBlockHeader.ProtocolVersion().Equal(ProtocolVersion) {
		errorMessage := "protocol version mismatch in transactions block header"
		s.logger.Error(errorMessage, log.Stringable("expected", ProtocolVersion), log.Stringable("received", txBlockHeader.ProtocolVersion()), log.BlockHeight(txBlockHeader.BlockHeight()))
		return fmt.Errorf(errorMessage)
	}

	if !rsBlockHeader.ProtocolVersion().Equal(ProtocolVersion) {
		errorMessage := "protocol version mismatch in results block header"
		s.logger.Error(errorMessage, log.Stringable("expected", ProtocolVersion), log.Stringable("received", rsBlockHeader.ProtocolVersion()), log.BlockHeight(txBlockHeader.BlockHeight()))
		return fmt.Errorf(errorMessage)
	}

	return nil
}

func (s *service) notifyConsensusAlgos(
	ctx context.Context,
	prevBlockPair *protocol.BlockPairContainer,
	blockPair *protocol.BlockPairContainer,
	mode handlers.HandleBlockConsensusMode) error {

	verifyMode := mode == handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE ||
		mode == handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_ONLY

	s.consensusBlocksHandlers.RLock()
	defer s.consensusBlocksHandlers.RUnlock()

	var verifyErrors []error
	verifiedCount := 0
	for _, handler := range s.consensusBlocksHandlers.handlers {
		_, latestErr := handler.HandleBlockConsensus(ctx, &handlers.HandleBlockConsensusInput{
			Mode:                   mode,
			BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
			BlockPair:              blockPair,
			PrevCommittedBlockPair: prevBlockPair, // TODO (v1) rename to HandleBlockConsensusInput.PrevCommittedBlockPair to PrevBlockPair
		})

		if verifyMode && latestErr == nil {
			verifiedCount++
		} else {
			verifyErrors = append(verifyErrors, latestErr)
		}
	}

	if verifyMode && verifiedCount == 0 && ctx.Err() == nil { // only log errors if system is not shutting down
		for _, err := range verifyErrors {
			s.logger.Error("consensus algo refused to validate block", log.Error(err))
		}
		return errors.Errorf("all consensus %d algos refused to validate the block", len(s.consensusBlocksHandlers.handlers))
	}

	return nil
}

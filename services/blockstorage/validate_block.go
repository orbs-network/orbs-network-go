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

// FIXME implement all block checks
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

	if blockHeightError := s.validateBlockHeight(input.BlockPair, lastCommittedBlock); blockHeightError != nil {
		return nil, blockHeightError
	}

	if err := s.validateWithConsensusAlgosWithMode(
		ctx,
		lastCommittedBlock,
		input.BlockPair,
		handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE); err != nil {

		logger.Error("block validation by consensus algo failed", log.Error(err))
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
		return false, errors.New(errorMessage)
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

func (s *service) validateBlockHeight(blockPair *protocol.BlockPairContainer, lastCommittedBlock *protocol.BlockPairContainer) error {
	expectedBlockHeight := getBlockHeight(lastCommittedBlock) + 1

	txBlockHeader := blockPair.TransactionsBlock.Header
	rsBlockHeader := blockPair.ResultsBlock.Header

	if txBlockHeader.BlockHeight() != rsBlockHeader.BlockHeight() {
		return fmt.Errorf("block pair height mismatch. transactions height is %d, results height is %d", txBlockHeader.BlockHeight(), rsBlockHeader.BlockHeight())
	}

	if txBlockHeader.BlockHeight() > expectedBlockHeight {
		return fmt.Errorf("block height is %d, expected %d", txBlockHeader.BlockHeight(), expectedBlockHeight)
	}

	return nil
}

func (s *service) validateProtocolVersion(blockPair *protocol.BlockPairContainer) error {
	txBlockHeader := blockPair.TransactionsBlock.Header
	rsBlockHeader := blockPair.ResultsBlock.Header

	// FIXME we may be logging twice, this should be fixed when handling the logging structured errors in logger issue
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

func (s *service) validateWithConsensusAlgos(
	ctx context.Context,
	prevBlockPair *protocol.BlockPairContainer,
	lastCommittedBlockPair *protocol.BlockPairContainer) error {

	return s.validateWithConsensusAlgosWithMode(ctx, prevBlockPair, lastCommittedBlockPair, handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY)
}

func (s *service) validateWithConsensusAlgosWithMode(
	ctx context.Context,
	prevBlockPair *protocol.BlockPairContainer,
	lastCommittedBlockPair *protocol.BlockPairContainer,
	mode handlers.HandleBlockConsensusMode) error {

	for _, handler := range s.consensusBlocksHandlers {
		_, err := handler.HandleBlockConsensus(ctx, &handlers.HandleBlockConsensusInput{
			Mode:                   mode,
			BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
			BlockPair:              lastCommittedBlockPair,
			PrevCommittedBlockPair: prevBlockPair,
		})

		// one of the consensus algos has validated the block, this means it's a valid block
		if err == nil {
			return nil
		}
	}

	return errors.Errorf("all consensus %d algos refused to validate the block", len(s.consensusBlocksHandlers))
}


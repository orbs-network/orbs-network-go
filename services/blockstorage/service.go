package blockstorage

import (
	"errors"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"sync"
	"time"
)

type Config interface {
	BlockSyncCommitTimeoutMillis() time.Duration
}

const (
	// TODO extract it to the spec
	ProtocolVersion = 1
)

type service struct {
	persistence  adapter.BlockPersistence
	stateStorage services.StateStorage

	config Config

	reporting               instrumentation.BasicLogger
	consensusBlocksHandlers []handlers.ConsensusBlocksHandler

	lastCommittedBlockHeight    primitives.BlockHeight
	lastCommittedBlockTimestamp primitives.TimestampNano
	lastBlockLock               *sync.Mutex
}

func NewBlockStorage(config Config, persistence adapter.BlockPersistence, stateStorage services.StateStorage, reporting instrumentation.BasicLogger) services.BlockStorage {
	return &service{
		persistence:   persistence,
		stateStorage:  stateStorage,
		reporting:     reporting.For(instrumentation.Service("block-storage")),
		config:        config,
		lastBlockLock: &sync.Mutex{},
	}
}

func (s *service) CommitBlock(input *services.CommitBlockInput) (*services.CommitBlockOutput, error) {
	txBlockHeader := input.BlockPair.TransactionsBlock.Header
	s.reporting.Info("Trying to commit a block", instrumentation.BlockHeight(txBlockHeader.BlockHeight()))

	if err := s.validateProtocolVersion(input.BlockPair); err != nil {
		return nil, err
	}

	// TODO there might be a non-idiomatic pattern here, but the commit block output is an empty struct, if that changes this should be refactored
	if ok, err := s.validateBlockDoesNotExist(txBlockHeader); err != nil || !ok {
		return nil, err
	}

	if err := s.validateBlockHeight(input.BlockPair); err != nil {
		return nil, err
	}

	if err := s.persistence.WriteBlock(input.BlockPair); err != nil {
		return nil, err
	}

	s.updateLastCommittedBlockHeightAndTimestamp(txBlockHeader)

	s.reporting.Info("Committed a block", instrumentation.BlockHeight(txBlockHeader.BlockHeight()))

	if err := s.syncBlockToStateStorage(input.BlockPair); err != nil {
		// TODO: since the intra-node sync flow is self healing, we should not fail the entire commit if state storage is slow to sync
		s.reporting.Error("intra-node sync to state storage failed", instrumentation.Error(err))
	}

	return nil, nil
}

func (s *service) updateLastCommittedBlockHeightAndTimestamp(txBlockHeader *protocol.TransactionsBlockHeader) {
	s.lastBlockLock.Lock()
	defer s.lastBlockLock.Unlock()

	s.lastCommittedBlockHeight = txBlockHeader.BlockHeight()
	s.lastCommittedBlockTimestamp = txBlockHeader.Timestamp()
}

func (s *service) loadTransactionsBlockHeader(height primitives.BlockHeight) (*services.GetTransactionsBlockHeaderOutput, error) {
	txBlock, err := s.persistence.GetTransactionsBlock(height)

	if err != nil {
		return nil, err
	}

	return &services.GetTransactionsBlockHeaderOutput{
		TransactionsBlockProof:    txBlock.BlockProof,
		TransactionsBlockHeader:   txBlock.Header,
		TransactionsBlockMetadata: txBlock.Metadata,
	}, nil
}

func (s *service) GetTransactionsBlockHeader(input *services.GetTransactionsBlockHeaderInput) (*services.GetTransactionsBlockHeaderOutput, error) {
	currentBlockHeight := s.lastCommittedBlockHeight
	if input.BlockHeight > currentBlockHeight && input.BlockHeight-currentBlockHeight <= 5 {
		// TODO try to remove this type of polling in the future: https://github.com/orbs-network/orbs-network-go/issues/54
		timeout := time.NewTimer(s.config.BlockSyncCommitTimeoutMillis())
		defer timeout.Stop()
		tick := time.NewTicker(10 * time.Millisecond)
		defer tick.Stop()

		for {
			select {
			case <-timeout.C:
				return nil, errors.New("operation timed out")
			case <-tick.C:
				if input.BlockHeight <= s.lastCommittedBlockHeight {
					lookupResult, err := s.loadTransactionsBlockHeader(input.BlockHeight)

					if err == nil {
						return lookupResult, nil
					}
				}
			}
		}
	}

	return s.loadTransactionsBlockHeader(input.BlockHeight)
}

func (s *service) loadResultsBlockHeader(height primitives.BlockHeight) (*services.GetResultsBlockHeaderOutput, error) {
	txBlock, err := s.persistence.GetResultsBlock(height)

	if err != nil {
		return nil, err
	}

	return &services.GetResultsBlockHeaderOutput{
		ResultsBlockProof:  txBlock.BlockProof,
		ResultsBlockHeader: txBlock.Header,
	}, nil
}

func (s *service) GetResultsBlockHeader(input *services.GetResultsBlockHeaderInput) (result *services.GetResultsBlockHeaderOutput, err error) {
	currentBlockHeight := s.lastCommittedBlockHeight
	if input.BlockHeight > currentBlockHeight && input.BlockHeight-currentBlockHeight <= 5 {
		timeout := time.NewTimer(s.config.BlockSyncCommitTimeoutMillis())
		defer timeout.Stop()
		tick := time.NewTicker(10 * time.Millisecond)
		defer tick.Stop()

		for {
			select {
			case <-timeout.C:
				return nil, errors.New("operation timed out")
			case <-tick.C:
				if input.BlockHeight <= s.lastCommittedBlockHeight {
					lookupResult, err := s.loadResultsBlockHeader(input.BlockHeight)

					if err == nil {
						return lookupResult, nil
					}
				}
			}
		}
	}

	return s.loadResultsBlockHeader(input.BlockHeight)
}

func (s *service) GetTransactionReceipt(input *services.GetTransactionReceiptInput) (*services.GetTransactionReceiptOutput, error) {
	panic("Not implemented")
}

func (s *service) GetLastCommittedBlockHeight(input *services.GetLastCommittedBlockHeightInput) (*services.GetLastCommittedBlockHeightOutput, error) {
	return &services.GetLastCommittedBlockHeightOutput{
		LastCommittedBlockHeight:    s.lastCommittedBlockHeight,
		LastCommittedBlockTimestamp: s.lastCommittedBlockTimestamp,
	}, nil
}

func (s *service) ValidateBlockForCommit(input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error) {
	if protocolVersionError := s.validateProtocolVersion(input.BlockPair); protocolVersionError != nil {
		return nil, protocolVersionError
	}

	if blockHeightError := s.validateBlockHeight(input.BlockPair); blockHeightError != nil {
		return nil, blockHeightError
	}

	return &services.ValidateBlockForCommitOutput{}, nil
}

func (s *service) RegisterConsensusBlocksHandler(handler handlers.ConsensusBlocksHandler) {
	s.consensusBlocksHandlers = append(s.consensusBlocksHandlers, handler)
}

func (s *service) HandleBlockAvailabilityRequest(input *gossiptopics.BlockAvailabilityRequestInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}

func (s *service) HandleBlockAvailabilityResponse(input *gossiptopics.BlockAvailabilityResponseInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}
func (s *service) HandleBlockSyncRequest(input *gossiptopics.BlockSyncRequestInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}
func (s *service) HandleBlockSyncResponse(input *gossiptopics.BlockSyncResponseInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}

//TODO how do we check if block with same height is the same block? do we compare the block bit-by-bit? https://github.com/orbs-network/orbs-spec/issues/50
func (s *service) validateBlockDoesNotExist(txBlockHeader *protocol.TransactionsBlockHeader) (bool, error) {
	currentBlockHeight := s.lastCommittedBlockHeight
	if txBlockHeader.BlockHeight() <= currentBlockHeight {
		if txBlockHeader.BlockHeight() == currentBlockHeight && txBlockHeader.Timestamp() != s.lastCommittedBlockTimestamp {
			errorMessage := "block already in storage, timestamp mismatch"
			s.reporting.Error(errorMessage, instrumentation.BlockHeight(currentBlockHeight))
			return false, errors.New(errorMessage)
		}

		s.reporting.Info("block already in storage, skipping", instrumentation.BlockHeight(currentBlockHeight))
		return false, nil
	}

	return true, nil
}

func (s *service) validateBlockHeight(blockPair *protocol.BlockPairContainer) error {
	expectedBlockHeight := s.lastCommittedBlockHeight + 1

	txBlockHeader := blockPair.TransactionsBlock.Header
	rsBlockHeader := blockPair.ResultsBlock.Header

	if txBlockHeader.BlockHeight() != expectedBlockHeight {
		return fmt.Errorf("block height is %d, expected %d", txBlockHeader.BlockHeight(), expectedBlockHeight)
	}

	if rsBlockHeader.BlockHeight() != expectedBlockHeight {
		return fmt.Errorf("block height is %d, expected %d", rsBlockHeader.BlockHeight(), expectedBlockHeight)
	}

	return nil
}

func (s *service) validateProtocolVersion(blockPair *protocol.BlockPairContainer) error {
	txBlockHeader := blockPair.TransactionsBlock.Header
	rsBlockHeader := blockPair.ResultsBlock.Header

	// FIXME we may be logging twice, this should be fixed when handling the logging structured errors in logger issue
	if txBlockHeader.ProtocolVersion() != ProtocolVersion {
		errorMessage := "protocol version mismatch"
		s.reporting.Error(errorMessage, instrumentation.String("expected", "1"), instrumentation.Stringable("received", txBlockHeader.ProtocolVersion()))
		return fmt.Errorf(errorMessage)
	}

	if rsBlockHeader.ProtocolVersion() != ProtocolVersion {
		errorMessage := "protocol version mismatch"
		s.reporting.Error(errorMessage, instrumentation.String("expected", "1"), instrumentation.Stringable("received", txBlockHeader.ProtocolVersion()))
		return fmt.Errorf(errorMessage)
	}

	return nil
}

// TODO: this should not be called directly from CommitBlock, it should be called from a long living goroutine that continuously syncs the state storage
func (s *service) syncBlockToStateStorage(commitedBlockPair *protocol.BlockPairContainer) error {
	_, err := s.stateStorage.CommitStateDiff(&services.CommitStateDiffInput{
		ResultsBlockHeader: commitedBlockPair.ResultsBlock.Header,
		ContractStateDiffs: commitedBlockPair.ResultsBlock.ContractStateDiffs,
	})
	return err
}

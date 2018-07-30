package blockstorage

import (
	"encoding/binary"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"time"
)

const (
	// TODO extract it to the spec
	ProtocolVersion          = 1
	NanosecondsInMillisecond = 1000000
)

type service struct {
	persistence  adapter.BlockPersistence
	stateStorage services.StateStorage

	config BlockStorageConfig

	lastCommittedBlockHeight    primitives.BlockHeight
	lastCommittedBlockTimestamp primitives.TimestampNano
	reporting                   instrumentation.BasicLogger
	consensusBlocksHandlers     []handlers.ConsensusBlocksHandler
}

func NewBlockStorage(config BlockStorageConfig, persistence adapter.BlockPersistence, stateStorage services.StateStorage, reporting instrumentation.BasicLogger) services.BlockStorage {
	return &service{
		persistence:  persistence,
		stateStorage: stateStorage,
		reporting:    reporting.For(instrumentation.Service("block-storage")),
		config:       config,
	}
}

func (s *service) CommitBlock(input *services.CommitBlockInput) (*services.CommitBlockOutput, error) {
	txBlockHeader := input.BlockPair.TransactionsBlock.Header
	s.reporting.Info("Trying to commit a block", instrumentation.BlockHeight(txBlockHeader.BlockHeight()))

	if err := s.validateProtocolVersion(input.BlockPair); err != nil {
		return nil, err
	}

	if ok := s.validateBlockDoesNotExist(txBlockHeader); !ok {
		return nil, nil
	}

	s.validateMonotonicIncreasingBlockHeight(txBlockHeader)

	s.persistence.WriteBlock(input.BlockPair)

	s.lastCommittedBlockHeight = txBlockHeader.BlockHeight()
	s.lastCommittedBlockTimestamp = txBlockHeader.Timestamp()

	// TODO: why are we updating the state? nothing about this in the spec
	s.updateStateStorage(input.BlockPair.TransactionsBlock)

	s.reporting.Info("Committed a block", instrumentation.BlockHeight(txBlockHeader.BlockHeight()))

	return nil, nil
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
	if input.BlockHeight > s.lastCommittedBlockHeight && input.BlockHeight-s.lastCommittedBlockHeight <= 5 {
		c := make(chan *services.GetTransactionsBlockHeaderOutput)

		go func() {
			const interval = 10 * NanosecondsInMillisecond
			timeout := s.config.BlockSyncCommitTimeout().Nanoseconds()

			for i := int64(0); i < timeout; i += interval {
				if input.BlockHeight <= s.lastCommittedBlockHeight {
					lookupResult, err := s.loadTransactionsBlockHeader(input.BlockHeight)

					if err == nil {
						c <- lookupResult
						return
					}
				}

				time.Sleep(10 * time.Millisecond)
			}

			c <- nil
		}()

		if result := <-c; result != nil {
			return result, nil
		}

		return nil, fmt.Errorf("operation timed out")
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
	if input.BlockHeight > s.lastCommittedBlockHeight && input.BlockHeight-s.lastCommittedBlockHeight <= 5 {
		c := make(chan *services.GetResultsBlockHeaderOutput)

		go func() {
			const interval = 10 * NanosecondsInMillisecond // 10 ms
			timeout := s.config.BlockSyncCommitTimeout().Nanoseconds()

			for i := int64(0); i < timeout; i += interval {
				if input.BlockHeight <= s.lastCommittedBlockHeight {
					lookupResult, err := s.loadResultsBlockHeader(input.BlockHeight)

					if err == nil {
						c <- lookupResult
						return
					}
				}

				time.Sleep(10 * time.Millisecond)
			}

			c <- nil
		}()

		if result := <-c; result != nil {
			return result, nil
		}

		return nil, fmt.Errorf("operation timed out")
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
func (s *service) validateBlockDoesNotExist(txBlockHeader *protocol.TransactionsBlockHeader) bool {
	if txBlockHeader.BlockHeight() <= s.lastCommittedBlockHeight {
		if txBlockHeader.BlockHeight() == s.lastCommittedBlockHeight && txBlockHeader.Timestamp() != s.lastCommittedBlockTimestamp {
			// TODO should this really panic
			errorMessage := "block already in storage, timestamp mismatch"
			s.reporting.Error(errorMessage, instrumentation.BlockHeight(s.lastCommittedBlockHeight))
			panic(errorMessage)
		}

		s.reporting.Info("block already in storage, skipping", instrumentation.BlockHeight(s.lastCommittedBlockHeight))
		return false
	}

	return true
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

	if txBlockHeader.ProtocolVersion() != ProtocolVersion {
		errorMessage := "protocol version mismatch"
		s.reporting.Error(errorMessage, instrumentation.String("expected", "1"), instrumentation.Stringable("received", txBlockHeader.ProtocolVersion()))
		return fmt.Errorf(errorMessage)
	}

	if rsBlockHeader.ProtocolVersion() != ProtocolVersion {
		return fmt.Errorf("protocol version mismatch: expected 1 got %d", rsBlockHeader.ProtocolVersion())
	}

	return nil
}

func (s *service) validateMonotonicIncreasingBlockHeight(txBlockHeader *protocol.TransactionsBlockHeader) {
	expectedNextBlockHeight := s.lastCommittedBlockHeight + 1
	if txBlockHeader.BlockHeight() != expectedNextBlockHeight {
		// TODO should this really panic
		errorMessage := "block height mismatch"
		s.reporting.Error(errorMessage, instrumentation.Stringable("expectedBlockHeight", expectedNextBlockHeight), instrumentation.Stringable("receivedBlockHeight", txBlockHeader.BlockHeight()))
		panic(fmt.Errorf(errorMessage))

	}
}

func (s *service) updateStateStorage(txBlock *protocol.TransactionsBlockContainer) {
	var state []*protocol.StateRecordBuilder
	for _, i := range txBlock.SignedTransactions {
		byteArray := make([]byte, 8)
		binary.LittleEndian.PutUint64(byteArray, uint64(i.Transaction().InputArgumentsIterator().NextInputArguments().Uint64Value()))
		transactionStateDiff := &protocol.StateRecordBuilder{
			Value: byteArray,
		}
		state = append(state, transactionStateDiff)
	}
	csdi := []*protocol.ContractStateDiff{(&protocol.ContractStateDiffBuilder{StateDiffs: state}).Build()}
	s.stateStorage.CommitStateDiff(&services.CommitStateDiffInput{ContractStateDiffs: csdi})
}

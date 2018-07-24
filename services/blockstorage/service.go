package blockstorage

import (
	"encoding/binary"
	"fmt"
	"time"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

const (
	// TODO extract it to the spec
	ProtocolVersion = 1
)

type service struct {
	persistence  adapter.BlockPersistence
	stateStorage services.StateStorage

	lastCommittedBlockHeight    primitives.BlockHeight
	lastCommittedBlockTimestamp primitives.TimestampNano
	reporting                   instrumentation.Reporting
	consensusBlocksHandlers     []handlers.ConsensusBlocksHandler
}

func NewBlockStorage(persistence adapter.BlockPersistence, stateStorage services.StateStorage, reporting instrumentation.Reporting) services.BlockStorage {
	return &service{
		persistence:  persistence,
		stateStorage: stateStorage,
		reporting:    reporting,
	}
}

func (s *service) CommitBlock(input *services.CommitBlockInput) (*services.CommitBlockOutput, error) {
	txBlockHeader := input.BlockPair.TransactionsBlock.Header
	s.reporting.Infof("Trying to commit block of height %d", txBlockHeader.BlockHeight())

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

	s.reporting.Infof("Committed block of height %d", txBlockHeader.BlockHeight())

	return nil, nil
}

func (s *service) lookForTransactionsBlockHeader(height primitives.BlockHeight) (*services.GetTransactionsBlockHeaderOutput, error) {
	txBlock, err := s.persistence.GetTransactionsBlock(height)

	if err != nil {
		return nil, err
	}

	return &services.GetTransactionsBlockHeaderOutput{
		TransactionsBlockProof: txBlock.BlockProof,
		TransactionsBlockHeader: txBlock.Header,
		TransactionsBlockMetadata: txBlock.Metadata,
	}, nil
}

func (s *service) GetTransactionsBlockHeader(input *services.GetTransactionsBlockHeaderInput) (*services.GetTransactionsBlockHeaderOutput, error) {
	if input.BlockHeight > s.lastCommittedBlockHeight && input.BlockHeight - s.lastCommittedBlockHeight <= 5 {
		c := make(chan *services.GetTransactionsBlockHeaderOutput)

		go func() {
			// TODO extract to a config
			const interval = 10
			const timeout = 10000

			for i:=0; i < timeout; i+= interval {
				if input.BlockHeight <= s.lastCommittedBlockHeight {
					lookupResult, err := s.lookForTransactionsBlockHeader(input.BlockHeight)

					if err == nil {
						c <- lookupResult
						return
					}
				}

				time.Sleep(interval)
			}
		}()

		result:= <-c
		return result, nil
	}

	return s.lookForTransactionsBlockHeader(input.BlockHeight)
}

func (s *service) lookForResultsBlockHeader(height primitives.BlockHeight) (*services.GetResultsBlockHeaderOutput, error) {
	txBlock, err := s.persistence.GetResultsBlock(height)

	if err != nil {
		return nil, err
	}

	return &services.GetResultsBlockHeaderOutput{
		ResultsBlockProof: txBlock.BlockProof,
		ResultsBlockHeader: txBlock.Header,
	}, nil
}

func (s *service) GetResultsBlockHeader(input *services.GetResultsBlockHeaderInput) (result *services.GetResultsBlockHeaderOutput, err error) {
	if input.BlockHeight > s.lastCommittedBlockHeight && input.BlockHeight - s.lastCommittedBlockHeight <= 5 {
		c := make(chan *services.GetResultsBlockHeaderOutput)

		go func() {
			// TODO extract to a config
			const interval = 10
			const timeout = 10000

			for i:=0; i < timeout; i+= interval {
				if input.BlockHeight <= s.lastCommittedBlockHeight {
					lookupResult, err := s.lookForResultsBlockHeader(input.BlockHeight)

					if err == nil {
						c <- lookupResult
						return
					}
				}

				time.Sleep(interval)
			}
		}()

		result:= <-c
		return result, nil
	}

	return s.lookForResultsBlockHeader(input.BlockHeight)
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
		return &services.ValidateBlockForCommitOutput{}, protocolVersionError
	}

	if blockHeightError := s.validateBlockHeight(input.BlockPair); blockHeightError != nil {
		return &services.ValidateBlockForCommitOutput{}, blockHeightError
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
			err := fmt.Errorf("block with height %d already in storage, timestamp mismatch", s.lastCommittedBlockHeight)
			s.reporting.Error(err)
			panic(err.Error())
		}

		s.reporting.Infof("block with height %d already in storage, skipping", s.lastCommittedBlockHeight)
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
		err := fmt.Errorf("protocol version mismatch: expected 1 got %d", txBlockHeader.ProtocolVersion())
		s.reporting.Error(err)
		return err
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
		err := fmt.Errorf("expected block of height %d but got %d", expectedNextBlockHeight, txBlockHeader.BlockHeight())
		s.reporting.Error(err)
		panic(err.Error())

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

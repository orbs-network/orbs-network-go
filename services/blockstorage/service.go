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
	reporting                   instrumentation.BasicLogger
	consensusBlocksHandlers     []handlers.ConsensusBlocksHandler
}

func NewBlockStorage(persistence adapter.BlockPersistence, stateStorage services.StateStorage, reporting instrumentation.BasicLogger) services.BlockStorage {
	return &service{
		persistence:  persistence,
		stateStorage: stateStorage,
		reporting:    reporting.For(instrumentation.Service("block-storage")),
	}
}

func (s *service) CommitBlock(input *services.CommitBlockInput) (*services.CommitBlockOutput, error) {
	txBlockHeader := input.BlockPair.TransactionsBlock.Header
	s.reporting.Info("Trying to commit a block", instrumentation.Int("blockHeight", int(txBlockHeader.BlockHeight())))

	if err := s.validateProtocolVersion(txBlockHeader); err != nil {
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

	s.reporting.Info("Committed a block", instrumentation.Int("blockHeight", int(txBlockHeader.BlockHeight())))

	return nil, nil
}

func (s *service) GetTransactionsBlockHeader(input *services.GetTransactionsBlockHeaderInput) (*services.GetTransactionsBlockHeaderOutput, error) {
	panic("Not implemented")
}

func (s *service) GetResultsBlockHeader(input *services.GetResultsBlockHeaderInput) (*services.GetResultsBlockHeaderOutput, error) {
	panic("Not implemented")
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
	panic("Not implemented")
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
			s.reporting.Error(errorMessage, instrumentation.Int("blockHeight", int(s.lastCommittedBlockHeight)))
			panic(errorMessage)
		}

		s.reporting.Info("block already in storage, skipping", instrumentation.Int("blockHeight", int(s.lastCommittedBlockHeight)))
		return false
	}

	return true
}

func (s *service) validateProtocolVersion(txBlockHeader *protocol.TransactionsBlockHeader) error {
	if txBlockHeader.ProtocolVersion() != ProtocolVersion {
		errorMessage := "protocol version mismatch"
		s.reporting.Error(errorMessage, instrumentation.Int("expected", 1), instrumentation.Int("received", int(txBlockHeader.ProtocolVersion())))
		return fmt.Errorf(errorMessage)
	}

	return nil
}

func (s *service) validateMonotonicIncreasingBlockHeight(txBlockHeader *protocol.TransactionsBlockHeader) {
	expectedNextBlockHeight := s.lastCommittedBlockHeight + 1
	if txBlockHeader.BlockHeight() != expectedNextBlockHeight {
		// TODO should this really panic
		errorMessage := "block height mismatch"
		s.reporting.Error(errorMessage, instrumentation.Int("expected", int(expectedNextBlockHeight)), instrumentation.Int("received", int(txBlockHeader.BlockHeight())))
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

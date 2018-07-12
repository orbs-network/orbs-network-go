package blockstorage

import (
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type service struct {
	services.BlockStorage
	persistence adapter.BlockPersistence
}

func NewBlockStorage(persistence adapter.BlockPersistence) services.BlockStorage {
	return &service{
		persistence: persistence,
	}
}

func (s *service) CommitBlock(input *services.CommitBlockInput) (*services.CommitBlockOutput, error) {
	for i := input.BlockPair.TransactionsBlock().SignedTransactionsOpaqueIterator(); i.HasNext(); {
		t := protocol.SignedTransactionReader(i.NextSignedTransactionsOpaque())
		if t.Transaction().InputArgumentsIterator().NextInputArguments().Uint64() > 1000{
				//TODO: handle invalid transaction gracefully
				return nil, nil
			}
	}
	s.persistence.WriteBlock(input.BlockPair)
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
	panic("Not implemented")
}

func (s *service) ValidateBlockForCommit(input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error) {
	panic("Not implemented")
}

func (s *service) RegisterConsensusBlocksHandler(handler handlers.ConsensusBlocksHandler) {
	panic("Not implemented")
}

func (s *service) HandleBlockAvailabilityRequest(input *gossiptopics.BlockSyncAvailabilityRequestInput) (*gossiptopics.BlockSyncOutput, error) {
	panic("Not implemented")
}
func (s *service) HandleBlockAvailabilityResponse(input *gossiptopics.BlockSyncAvailabilityResponseInput) (*gossiptopics.BlockSyncOutput, error) {
	panic("Not implemented")
}
func (s *service) HandleBlockSyncRequest(input *gossiptopics.BlockSyncRequestInput) (*gossiptopics.BlockSyncOutput, error) {
	panic("Not implemented")
}
func (s *service) HandleBlockSyncResponse(input *gossiptopics.BlockSyncResponseInput) (*gossiptopics.BlockSyncOutput, error) {
	panic("Not implemented")
}
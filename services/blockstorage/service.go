package blockstorage

import (
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/orbs-network/orbs-spec/types/go/services/gossip"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
)

func NewBlockStorage(persistence adapter.BlockPersistence) services.BlockStorage {
	return &service{persistence:persistence}
}

type service struct {
	persistence adapter.BlockPersistence
}

func (s *service) CommitBlock(input *services.CommitBlockInput) (*services.CommitBlockOutput, error) {
	panic("Not implemented")
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

func (s *service) HandleBlockAvailabilityRequest(input *gossip.BlockSyncAvailabilityRequestInput) (*gossip.BlockSyncOutput, error) {
	panic("Not implemented")
}
func (s *service) HandleBlockAvailabilityResponse(input *gossip.BlockSyncAvailabilityResponseInput) (*gossip.BlockSyncOutput, error) {
	panic("Not implemented")
}
func (s *service) HandleBlockSyncRequest(input *gossip.BlockSyncRequestInput) (*gossip.BlockSyncOutput, error) {
	panic("Not implemented")
}
func (s *service) HandleBlockSyncResponse(input *gossip.BlockSyncResponseInput) (*gossip.BlockSyncOutput, error) {
	panic("Not implemented")
}

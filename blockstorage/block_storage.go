package blockstorage

import (
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

func NewBlockStorage(persistence BlockPersistence) services.BlockStorage {
	return &blockStorage{persistence:persistence}
}

type blockStorage struct {
	persistence BlockPersistence
}

func (b *blockStorage) CommitBlock(input *services.CommitBlockInput) (*services.CommitBlockOutput, error) {
	panic("Not implemented")
}

func (b *blockStorage) GetTransactionsBlockHeader(input *services.GetTransactionsBlockHeaderInput) (*services.GetTransactionsBlockHeaderOutput, error) {
	panic("Not implemented")
}

func (b *blockStorage) GetResultsBlockHeader(input *services.GetResultsBlockHeaderInput) (*services.GetResultsBlockHeaderOutput, error) {
	panic("Not implemented")
}

func (b *blockStorage) GetTransactionReceipt(input *services.GetTransactionReceiptInput) (*services.GetTransactionReceiptOutput, error) {
	panic("Not implemented")
}

func (b *blockStorage) GetLastCommittedBlockHeight(input *services.GetLastCommittedBlockHeightInput) (*services.GetLastCommittedBlockHeightOutput, error) {
	panic("Not implemented")
}

func (b *blockStorage) ValidateBlockForCommit(input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error) {
	panic("Not implemented")
}

func (b *blockStorage) RegisterConsensusBlocksHandler(handler handlers.ConsensusBlocksHandler) {
	panic("Not implemented")
}

func (b *blockStorage) HandleBlockAvailabilityRequest(input *handlers.HandleBlockAvailabilityRequestInput) (*handlers.GossipMessageHandlerOutput, error) {
	panic("Not implemented")
}

func (b *blockStorage) HandleBlockAvailabilityResponse(input *handlers.HandleBlockAvailabilityResponseInput) (*handlers.GossipMessageHandlerOutput, error) {
	panic("Not implemented")
}

func (b *blockStorage) HandleBlockSyncRequest(input *handlers.HandleBlockSyncRequestInput) (*handlers.GossipMessageHandlerOutput, error) {
	panic("Not implemented")
}

func (b *blockStorage) HandleBlockSyncResponse(input *handlers.HandleBlockSyncResponseInput) (*handlers.GossipMessageHandlerOutput, error) {
	panic("Not implemented")
}

package sync

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
)

type processingBlocksState struct {
	blocks *gossipmessages.BlockSyncResponseMessage
	sf     *stateFactory
}

func (s *processingBlocksState) name() string {
	return "processing-blocks-state"
}

func (s *processingBlocksState) processState(ctx context.Context) syncState {
	panic("implement me")
}

func (s *processingBlocksState) blockCommitted(blockHeight primitives.BlockHeight) {
	panic("implement me")
}

func (s *processingBlocksState) gotAvailabilityResponse(message *gossipmessages.BlockAvailabilityResponseMessage) {
	panic("implement me")
}

func (s *processingBlocksState) gotBlocks(source primitives.Ed25519PublicKey, message *gossipmessages.BlockSyncResponseMessage) {
	panic("implement me")
}

package sync

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
)

type waitingForChunksState struct {
	sf        *stateFactory
	sourceKey primitives.Ed25519PublicKey
}

func (s *waitingForChunksState) name() string {
	return "waiting-for-chunks-state"
}

func (s *waitingForChunksState) processState(ctx context.Context) syncState {
	panic("implement me")
}

func (s *waitingForChunksState) blockCommitted(blockHeight primitives.BlockHeight) {
	panic("implement me")
}

func (s *waitingForChunksState) gotAvailabilityResponse(message *gossipmessages.BlockAvailabilityResponseMessage) {
	panic("implement me")
}

func (s *waitingForChunksState) gotBlocks(source primitives.Ed25519PublicKey, blocks []*protocol.BlockPairContainer) {
	panic("implement me")
}

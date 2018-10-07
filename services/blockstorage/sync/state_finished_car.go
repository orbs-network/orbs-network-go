package sync

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
)

type finishedCARState struct{}

func (finishedCARState) name() string {
	return "finished-collecting-availability-requests-state"
}

func (finishedCARState) processState(ctx context.Context) syncState {
	panic("implement me")
}

func (finishedCARState) blockCommitted(blockHeight primitives.BlockHeight) {
	panic("implement me")
}

func (finishedCARState) gotAvailabilityResponse(message *gossipmessages.BlockAvailabilityResponseMessage) {
	panic("implement me")
}

func (finishedCARState) gotBlocks(source primitives.Ed25519PublicKey, blocks []*protocol.BlockPairContainer) {
	panic("implement me")
}

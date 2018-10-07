package sync

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
)

type finishedCARState struct {
	responses []*gossipmessages.BlockAvailabilityResponseMessage
	sf        *stateFactory
}

func (s *finishedCARState) name() string {
	return "finished-collecting-availability-requests-state"
}

func (s *finishedCARState) processState(ctx context.Context) syncState {
	if len(s.responses) == 0 {
		return s.sf.CreateIdleState()
	}

	return nil
}

func (s *finishedCARState) blockCommitted(blockHeight primitives.BlockHeight) {
	panic("implement me")
}

func (s *finishedCARState) gotAvailabilityResponse(message *gossipmessages.BlockAvailabilityResponseMessage) {
	panic("implement me")
}

func (s *finishedCARState) gotBlocks(source primitives.Ed25519PublicKey, blocks []*protocol.BlockPairContainer) {
	panic("implement me")
}

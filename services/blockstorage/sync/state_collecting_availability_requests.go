package sync

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
)

type collectingAvailabilityResponsesState struct{}

func (s *collectingAvailabilityResponsesState) name() string {
	return "collecting-availability-responses"
}

func (s *collectingAvailabilityResponsesState) next() syncState {
	panic("implement me")
}

func (s *collectingAvailabilityResponsesState) blockCommitted(blockHeight primitives.BlockHeight) {
	panic("implement me")
}

func (s *collectingAvailabilityResponsesState) gotAvailabilityResponse(message gossipmessages.BlockAvailabilityResponseMessage) {
	panic("implement me")
}

func (s *collectingAvailabilityResponsesState) gotBlocks(source primitives.Ed25519PublicKey, blocks []*protocol.BlockPairContainer) {
	panic("implement me")
}

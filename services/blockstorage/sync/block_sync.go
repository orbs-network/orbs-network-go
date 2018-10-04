package sync

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
)

type syncState interface {
	next() syncState
	blockCommitted(blockHeight primitives.BlockHeight)
	gotAvailabilityResponse(message gossipmessages.BlockAvailabilityResponseMessage)
	gotBlocks(source primitives.Ed25519PublicKey, blocks []*protocol.BlockPairContainer)
}

type blockSync struct {
	lastBlockHeight primitives.BlockHeight
}

func NewBlockSync(bh primitives.BlockHeight) *blockSync {
	return &blockSync{
		lastBlockHeight: bh,
	}
}

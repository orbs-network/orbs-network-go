package test

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

func (h *harness) handleBlockConsensus(blockPair *protocol.BlockPairContainer, prevCommitted *protocol.BlockPairContainer) error {
	_, err := h.service.HandleBlockConsensus(&handlers.HandleBlockConsensusInput{
		BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
		BlockPair:              blockPair,
		PrevCommittedBlockPair: prevCommitted,
	})
	return err
}

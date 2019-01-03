package leanhelixconsensus

import (
	"context"
	lh "github.com/orbs-network/lean-helix-go/services/interfaces"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
)

func (p *blockProvider) ValidateBlockProposal(ctx context.Context, blockHeight lhprimitives.BlockHeight, block lh.Block, blockHash lhprimitives.BlockHash, prevBlock lh.Block) bool {
	// TODO Implement me

	return true
}

func (p *blockProvider) ValidateBlockCommitment(blockHeight lhprimitives.BlockHeight, block lh.Block, blockHash lhprimitives.BlockHash) bool {
	// TODO Implement me

	return true
}

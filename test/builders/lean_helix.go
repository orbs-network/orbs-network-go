package builders

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
)

// protocol.BlockPairContainer

func LeanHelixBlockPair() *blockPair {
	return BlockPair().WithLeanHelixBlockProof()
}

func (b *blockPair) WithLeanHelixBlockProof() *blockPair {
	b.txProof = &protocol.TransactionsBlockProofBuilder{
		Type:      protocol.TRANSACTIONS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		LeanHelix: &consensus.LeanHelixBlockProofBuilder{},
	}
	b.rxProof = &protocol.ResultsBlockProofBuilder{
		Type:      protocol.RESULTS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		LeanHelix: &consensus.LeanHelixBlockProofBuilder{},
	}
	return b
}

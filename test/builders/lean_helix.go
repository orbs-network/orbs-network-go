package builders

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

// protocol.BlockPairContainer

func LeanHelixBlockPair() *blockPair {
	return BlockPair().WithLeanHelixBlockProof()
}

// TODO (v1) Fix with correct block proof
func (b *blockPair) WithLeanHelixBlockProof() *blockPair {
	b.txProof = &protocol.TransactionsBlockProofBuilder{
		Type:      protocol.TRANSACTIONS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		LeanHelix: nil,
	}
	b.rxProof = &protocol.ResultsBlockProofBuilder{
		Type:      protocol.RESULTS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		LeanHelix: nil,
	}
	return b
}

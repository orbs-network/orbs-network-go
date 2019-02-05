package builders

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

func (b *blockPair) WithEmptyLeanHelixBlockProof() *blockPair {
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

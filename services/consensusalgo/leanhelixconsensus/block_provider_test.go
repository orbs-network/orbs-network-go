package leanhelixconsensus

import (
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestValidLeanHelixBlockPair(t *testing.T) {

	leanHelixProof1 := primitives.LeanHelixBlockProof(hash.CalcSha256([]byte{1, 2, 3, 4}))
	leanHelixProof2 := primitives.LeanHelixBlockProof(hash.CalcSha256([]byte{5, 6, 7, 8}))

	txBlockProof1 := (&protocol.TransactionsBlockProofBuilder{
		ResultsBlockHash: nil,
		Type:             protocol.TRANSACTIONS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		LeanHelix:        leanHelixProof1,
	}).Build()

	rxBlockProof1 := (&protocol.ResultsBlockProofBuilder{
		TransactionsBlockHash: nil,
		Type:                  protocol.RESULTS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		LeanHelix:             leanHelixProof1,
	}).Build()

	txBlockProof2 := (&protocol.TransactionsBlockProofBuilder{
		ResultsBlockHash: nil,
		Type:             protocol.TRANSACTIONS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		LeanHelix:        leanHelixProof2,
	}).Build()
	txBlock := &(protocol.TransactionsBlockContainer{
		Header:             nil,
		Metadata:           nil,
		SignedTransactions: nil,
		BlockProof:         txBlockProof1,
	})
	rxBlock := &protocol.ResultsBlockContainer{
		Header:              nil,
		TransactionReceipts: nil,
		ContractStateDiffs:  nil,
		BlockProof:          rxBlockProof1,
	}

	t.Run("error on nil", func(t *testing.T) {
		err := validLeanHelixBlockPair(nil)
		require.Error(t, err)
	})
	t.Run("error on nil block proof in either transactions or results block", func(t *testing.T) {
		err := validLeanHelixBlockPair(&protocol.BlockPairContainer{
			TransactionsBlock: nil,
			ResultsBlock:      rxBlock,
		})
		require.Error(t, err)
		err = validLeanHelixBlockPair(&protocol.BlockPairContainer{
			TransactionsBlock: txBlock,
			ResultsBlock:      nil,
		})
		require.Error(t, err)
		err = validLeanHelixBlockPair(&protocol.BlockPairContainer{
			TransactionsBlock: nil,
			ResultsBlock:      nil,
		})
		require.Error(t, err)
	})
	t.Run("error on non-leanhelix block proof in either transactions or results block", func(t *testing.T) {
		validBlock := &protocol.BlockPairContainer{
			TransactionsBlock: txBlock,
			ResultsBlock:      rxBlock,
		}
		validBlock.TransactionsBlock.BlockProof.MutateLeanHelix(nil)
		validBlock.ResultsBlock.BlockProof.MutateLeanHelix(nil)

		err := validLeanHelixBlockPair(nil)
		require.Error(t, err)

	})
	t.Run("error on different blockproofs in transactions and results block", func(t *testing.T) {
		validBlock := &protocol.BlockPairContainer{
			TransactionsBlock: txBlock,
			ResultsBlock:      rxBlock,
		}
		validBlock.TransactionsBlock.BlockProof = txBlockProof2
		err := validLeanHelixBlockPair(nil)
		require.Error(t, err)
		validBlock.TransactionsBlock.BlockProof = txBlockProof1 // so that the next test won't fail
	})
	t.Run("ok on valid block", func(t *testing.T) {
		validBlock := &protocol.BlockPairContainer{
			TransactionsBlock: txBlock,
			ResultsBlock:      rxBlock,
		}
		err := validLeanHelixBlockPair(validBlock)
		require.Nil(t, err)
	})

}

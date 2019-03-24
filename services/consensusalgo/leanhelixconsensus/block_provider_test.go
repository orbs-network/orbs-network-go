// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package leanhelixconsensus

import (
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

type testBlock struct {
	block *protocol.BlockPairContainer
}

func NewTestBlock() *testBlock {
	txBlockProof1 := (&protocol.TransactionsBlockProofBuilder{
		ResultsBlockHash: nil,
		Type:             protocol.TRANSACTIONS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		LeanHelix:        primitives.LeanHelixBlockProof(hash.CalcSha256([]byte{1, 2, 3, 4})),
	}).Build()

	rxBlockProof1 := (&protocol.ResultsBlockProofBuilder{
		TransactionsBlockHash: nil,
		Type:                  protocol.RESULTS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		LeanHelix:             primitives.LeanHelixBlockProof(hash.CalcSha256([]byte{1, 2, 3, 4})),
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
	return &testBlock{
		block: &protocol.BlockPairContainer{
			TransactionsBlock: txBlock,
			ResultsBlock:      rxBlock,
		},
	}
}

func (t *testBlock) withNilTransactionBlock() *testBlock {
	t.block.TransactionsBlock = nil
	return t
}

func (t *testBlock) withNilResultsBlock() *testBlock {
	t.block.ResultsBlock = nil
	return t
}

func (t *testBlock) withNilBlockProof() *testBlock {
	t.block.TransactionsBlock.BlockProof = nil
	t.block.ResultsBlock.BlockProof = nil
	return t
}

func (t *testBlock) build() *protocol.BlockPairContainer {
	return t.block
}

func (t *testBlock) withDifferentTxAndRxBlockProofs() *testBlock {
	t.block.TransactionsBlock.BlockProof.MutateLeanHelix(primitives.LeanHelixBlockProof(hash.CalcSha256([]byte{1, 2, 3, 4})))
	t.block.ResultsBlock.BlockProof.MutateLeanHelix(primitives.LeanHelixBlockProof(hash.CalcSha256([]byte{5, 6, 7, 8})))
	return t
}

func TestValidLeanHelixBlockPair_SuccessOnValidBlock(t *testing.T) {
	require.Nil(t, validLeanHelixBlockPair(NewTestBlock().build()), "should return nil on valid block")
}

func TestValidLeanHelixBlockPair_FailOnNil(t *testing.T) {

	require.Error(t, validLeanHelixBlockPair(nil), "should return error when block is nil")
	require.Error(t, validLeanHelixBlockPair(NewTestBlock().withNilTransactionBlock().withNilResultsBlock().build()), "should return error when txblock and rxblock are nil")
	require.Error(t, validLeanHelixBlockPair(NewTestBlock().withNilTransactionBlock().build()), "should return error when txblock is nil")
	require.Error(t, validLeanHelixBlockPair(NewTestBlock().withNilResultsBlock().build()), "should return error when rxblock is nil")
}

func TestValidLeanHelixBlockPair_FailOnNilBlockProof(t *testing.T) {
	require.Error(t, validLeanHelixBlockPair(NewTestBlock().withNilBlockProof().build()), "should return error when block proof is nil")
}

func TestValidLeanHelixBlockPair_FailOnDifferentBlockProofsBetweenTransactionsAndResultsBlocks(t *testing.T) {
	require.Error(t, validLeanHelixBlockPair(NewTestBlock().withDifferentTxAndRxBlockProofs().build()), "should return error if block proofs in transactions and results block are different")
}

// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"bytes"
	lhbuilders "github.com/orbs-network/lean-helix-go/test/builders"
	"github.com/orbs-network/orbs-network-go/services/consensusalgo/leanhelixconsensus"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBlockProofSerialization(t *testing.T) {
	expectedBlockProof := lhbuilders.AMockBlockProof().Raw()
	block := builders.BlockPair().Build()
	block.TransactionsBlock.BlockProof = leanhelixconsensus.CreateTransactionBlockProof(block, expectedBlockProof)
	actualBlockProof, err := leanhelixconsensus.ExtractBlockProof(block)
	require.NoError(t, err, "ExtractBlockProof should succeed if block contains LeanHelix proof")
	require.True(t, bytes.Equal(expectedBlockProof, actualBlockProof), "block proofs should be the same: expectedBlockProof=%v actualBlockProof=%v", expectedBlockProof, actualBlockProof)
}

func TestBlockProofSerialization_IncorrectExtraction(t *testing.T) {
	expectedBlockProof := lhbuilders.AMockBlockProof().Raw()
	block := builders.BlockPair().Build()
	block.TransactionsBlock.BlockProof = leanhelixconsensus.CreateTransactionBlockProof(block, expectedBlockProof)
	incorrectBlockProof := block.TransactionsBlock.BlockProof.Raw()
	require.False(t, bytes.Equal(expectedBlockProof, incorrectBlockProof), "block proofs should be different")
}

func TestExtractBlockProofFailsIfNotLeanHelixBlock(t *testing.T) {
	block := builders.BlockPair().Build()
	block.TransactionsBlock.BlockProof = (&protocol.TransactionsBlockProofBuilder{
		Type:               protocol.TRANSACTIONS_BLOCK_PROOF_TYPE_BENCHMARK_CONSENSUS,
		BenchmarkConsensus: &consensus.BenchmarkConsensusBlockProofBuilder{},
	}).Build()
	_, err := leanhelixconsensus.ExtractBlockProof(block)
	require.Error(t, err, "ExtractBlockProof should fail if block does not contain LeanHelix proof")
}

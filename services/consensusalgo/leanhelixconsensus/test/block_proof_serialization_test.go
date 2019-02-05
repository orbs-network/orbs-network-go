package test

import (
	"bytes"
	lhbuilders "github.com/orbs-network/lean-helix-go/test/builders"
	"github.com/orbs-network/orbs-network-go/services/consensusalgo/leanhelixconsensus"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBlockProofSerialization(t *testing.T) {
	expectedBlockProof := lhbuilders.AMockBlockProof().Raw()
	block := builders.BlockPair().Build()
	block.TransactionsBlock.BlockProof = leanhelixconsensus.CreateTransactionBlockProof(block, expectedBlockProof)
	actualBlockProof := leanhelixconsensus.ExtractBlockProof(block)
	require.True(t, bytes.Equal(expectedBlockProof, actualBlockProof), "block proofs should be the same: expectedBlockProof=%v actualBlockProof=%v", expectedBlockProof, actualBlockProof)
}

func TestBlockProofSerialization_IncorrectExtraction(t *testing.T) {
	expectedBlockProof := lhbuilders.AMockBlockProof().Raw()
	block := builders.BlockPair().Build()
	block.TransactionsBlock.BlockProof = leanhelixconsensus.CreateTransactionBlockProof(block, expectedBlockProof)
	incorrectBlockProof := block.TransactionsBlock.BlockProof.Raw()
	require.False(t, bytes.Equal(expectedBlockProof, incorrectBlockProof), "block proofs should be different")
}

package builders

import (
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

func RandomizedBlockChain(numBlocks int32, ctrlRand *rand.ControlledRand) []*protocol.BlockPairContainer {
	blocks := make([]*protocol.BlockPairContainer, 0, numBlocks)

	var prev *protocol.BlockPairContainer
	for bi := 1; bi <= cap(blocks); bi++ {
		newBlock := RandomizedBlock(primitives.BlockHeight(bi), ctrlRand, prev)
		blocks = append(blocks, newBlock)
		prev = newBlock
	}
	return blocks
}

func RandomizedBlock(h primitives.BlockHeight, ctrlRand *rand.ControlledRand, prev *protocol.BlockPairContainer) *protocol.BlockPairContainer {
	builder := BlockPair().
		WithHeight(h).
		WithTransactions(ctrlRand.Uint32() % 200).
		WithStateDiffs(ctrlRand.Uint32() % 200).
		WithReceiptsForTransactions().
		WithEmptyLeanHelixBlockProof()
	if prev != nil {
		builder.WithPrevBlock(prev)
	}
	return builder.Build()
}

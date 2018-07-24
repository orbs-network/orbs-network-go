package test

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

func CommitStateDiff() *commitStateDiffInputBuilder {
	return &commitStateDiffInputBuilder{
		headerBuilder: &protocol.ResultsBlockHeaderBuilder{},
	}
}

type commitStateDiffInputBuilder struct {
	headerBuilder *protocol.ResultsBlockHeaderBuilder
	diffs         []*protocol.ContractStateDiff
}

func (b *commitStateDiffInputBuilder) WithBlockHeight(h int) *commitStateDiffInputBuilder {
	b.headerBuilder.BlockHeight = primitives.BlockHeight(h)
	return b
}

func (b *commitStateDiffInputBuilder) WithDiff(diff *protocol.ContractStateDiff) *commitStateDiffInputBuilder {
	b.diffs = append(b.diffs, diff)
	return b
}

func (b *commitStateDiffInputBuilder) Build() *services.CommitStateDiffInput {
	return &services.CommitStateDiffInput{
		ResultsBlockHeader: b.headerBuilder.Build(),
		ContractStateDiffs: b.diffs,
	}
}
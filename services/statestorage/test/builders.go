// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
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

func (b *commitStateDiffInputBuilder) WithBlockTimestamp(t int) *commitStateDiffInputBuilder {
	b.headerBuilder.Timestamp = primitives.TimestampNano(t)
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

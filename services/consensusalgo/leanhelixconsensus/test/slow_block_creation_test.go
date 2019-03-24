// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

// This test shows the shy leader problem, that when we sync in lean helix, the petitioner
// thinks he is the leader on v=0 of old blocks and tries to propose a block.
// This causes it to get stuck on GetTransactionsForOrdering (9 seconds when no traffic)
// and broadcast large pre prepares that nobody cares about to everybody (network pollution).

func TestService_SlowBlockCreationDoesNotHurtBlockSync(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeanHelixServiceHarness().start(t, ctx)

		isLeanHelixStuckOnCreatingABlock := false

		h.expectConsensusContextRequestOrderingCommittee(0) // we're index 0 (first time called)
		h.expectConsensusContextRequestOrderingCommittee(0) // we're index 0 (second time called)
		h.consensusContextRespondWithVerySlowBlockCreation(&isLeanHelixStuckOnCreatingABlock)

		b5 := builders.BlockPair().WithHeight(5).WithEmptyLeanHelixBlockProof().Build()
		h.consensus.HandleBlockConsensus(ctx, &handlers.HandleBlockConsensusInput{
			Mode:                   handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY,
			BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
			BlockPair:              b5,
			PrevCommittedBlockPair: nil,
		})

		b6 := builders.BlockPair().WithHeight(6).WithEmptyLeanHelixBlockProof().Build()
		h.consensus.HandleBlockConsensus(ctx, &handlers.HandleBlockConsensusInput{
			Mode:                   handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY,
			BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
			BlockPair:              b6,
			PrevCommittedBlockPair: nil,
		})

		require.NoError(t, test.EventuallyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT, h.consensusContext))
		require.True(t, test.Consistently(test.CONSISTENTLY_ACCEPTANCE_TIMEOUT, func() bool {
			return !isLeanHelixStuckOnCreatingABlock
		}), "Lean Helix should not be creating a block and getting stuck doing so")
	})
}

func (h *harness) consensusContextRespondWithVerySlowBlockCreation(isCreatingABlock *bool) {
	h.consensusContext.When("RequestNewTransactionsBlock", mock.Any, mock.Any).Call(func(ctx context.Context, input *services.RequestNewTransactionsBlockInput) (*services.RequestNewTransactionsBlockOutput, error) {
		*isCreatingABlock = true
		<-ctx.Done()
		return nil, errors.New("canceled")
	})
}

// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/stretchr/testify/require"
	"testing"
)

// This test shows the shy leader problem, that when we sync in lean helix, the petitioner
// thinks he is the leader on v=0 of old blocks and tries to propose a block.
// This causes it to get stuck on GetTransactionsForOrdering (9 seconds when no traffic)
// and broadcast large pre prepares that nobody cares about to everybody (network pollution).

func TestService_ShyLeader_LeaderElectedByBlockSyncDoesNotProposeABlockOnFirstView(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeanHelixServiceHarness(0).start(t, ctx)

		h.expectConsensusContextRequestNewBlockNotCalled()

		h.expectConsensusContextRequestOrderingCommittee(0) // we're index 0 (first time called)

		b5 := builders.BlockPair().WithHeight(5).WithEmptyLeanHelixBlockProof().Build()
		_, _ = h.consensus.HandleBlockConsensus(ctx, &handlers.HandleBlockConsensusInput{
			Mode:                   handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY,
			BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
			BlockPair:              b5,
			PrevCommittedBlockPair: nil,
		})

		require.NoError(t, test.EventuallyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT, h.consensusContext), "expected ordering committee to be requested once")
		require.NoError(t, test.ConsistentlyVerify(test.CONSISTENTLY_ACCEPTANCE_TIMEOUT, h.consensusContext), "expected new block not to be requested by lean helix")

		h.resetAndApplyMockDefaults()
		h.expectConsensusContextRequestOrderingCommittee(0)

		b6 := builders.BlockPair().WithHeight(6).WithEmptyLeanHelixBlockProof().Build()
		_, _ = h.consensus.HandleBlockConsensus(ctx, &handlers.HandleBlockConsensusInput{
			Mode:                   handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY,
			BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
			BlockPair:              b6,
			PrevCommittedBlockPair: nil,
		})

		require.NoError(t, test.EventuallyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT, h.consensusContext), "expected ordering committee to be requested twice")
		require.NoError(t, test.ConsistentlyVerify(test.CONSISTENTLY_ACCEPTANCE_TIMEOUT, h.consensusContext), "expected new block not to be requested by lean helix")
	})
}

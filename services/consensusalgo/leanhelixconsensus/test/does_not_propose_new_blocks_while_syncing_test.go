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
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/stretchr/testify/require"
	"testing"
)

// This test shows the shy leader problem, that when we sync in lean helix, the petitioner
// might think it is the leader on v=0 of old blocks during block sync, and try to propose a block.
// This causes it to get block it's own sync flow if tx pool is empty (waiting 9 seconds for an empty block)
// and to perform meaningless work and broadcast large pre-prepares nobody cares about (network pollution).
// This test see that it does not attempt to propose a block after sync

func TestService_DoesNotProposeNewBlocksWhileSyncingBlocksSequentially(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeanHelixServiceHarness(0).start(t, ctx)

		syncFromBlock := primitives.BlockHeight(5)
		syncUpToBlock := primitives.BlockHeight(7) // exercise 3 block syncs in succession

		for currentHeight := syncFromBlock; currentHeight <= syncUpToBlock; currentHeight++ {

			h.resetAndApplyMockDefaults()
			h.expectNeverToProposeABlock()
			h.beFirstInCommittee() // we will be leader after sync:

			block := builders.BlockPair().WithHeight(currentHeight).WithEmptyLeanHelixBlockProof().Build()

			_, _ = h.consensus.HandleBlockConsensus(ctx, &handlers.HandleBlockConsensusInput{
				Mode:                   handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY,
				BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
				BlockPair:              block,
				PrevCommittedBlockPair: nil,
			})

			require.NoError(t, test.EventuallyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT, h.consensusContext), "expected ordering committee to be requested to determine next leader")
			require.NoError(t, test.ConsistentlyVerify(test.CONSISTENTLY_ACCEPTANCE_TIMEOUT, h.consensusContext), "expected new block not to be requested by lean helix")
		}
	})
}

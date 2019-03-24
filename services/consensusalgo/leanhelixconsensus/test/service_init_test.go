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

func TestService_StartsActivityOnlyAfterHandleBlockConsensus(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeanHelixServiceHarness()

		t.Log("Service should do nothing on start")

		h.expectConsensusContextRequestOrderingCommitteeNotCalled()

		h.start(t, ctx)

		err := test.ConsistentlyVerify(test.CONSISTENTLY_ACCEPTANCE_TIMEOUT, h.consensusContext)
		require.NoError(t, err)

		t.Log("Service should request committee after HandleBlockConsensus is called")

		h.expectConsensusContextRequestOrderingCommittee(1) // we're index 0

		h.consensus.HandleBlockConsensus(ctx, &handlers.HandleBlockConsensusInput{
			Mode:                   handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY,
			BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
			BlockPair:              nil,
			PrevCommittedBlockPair: nil,
		})

		require.NoError(t, test.EventuallyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT, h.consensusContext))
	})
}

func TestService_LeaderProposesBlock(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeanHelixServiceHarness().start(t, ctx)

		b := builders.BlockPair().WithEmptyLeanHelixBlockProof().Build()
		h.expectConsensusContextRequestOrderingCommittee(0) // we're index 0
		h.expectConsensusContextRequestBlock(b)
		h.expectGossipSendLeanHelixMessage()

		h.consensus.HandleBlockConsensus(ctx, &handlers.HandleBlockConsensusInput{
			Mode:                   handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY,
			BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
			BlockPair:              nil,
			PrevCommittedBlockPair: nil,
		})

		require.NoError(t, test.EventuallyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT, h.consensusContext, h.gossip))
	})
}

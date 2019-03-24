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
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"testing"
)

func TestHandlerOfLeaderSynchronizesToFutureValidBlock(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessWaitingForCommittedMessages(t, ctx)
		aBlockFromLeader := builders.BlockPair().WithBenchmarkConsensusBlockProof(leaderKeyPair())

		t.Log("Handle block consensus (ie due to block sync) of height 1002")

		b1001 := aBlockFromLeader.WithHeight(1001).Build()
		b1002 := aBlockFromLeader.WithHeight(1002).WithPrevBlock(b1001).Build()
		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(1002, h.config.NodeAddress())

		err := h.handleBlockConsensus(ctx, handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE, b1002, b1001)
		if err != nil {
			t.Fatal("handle did not validate valid block:", err)
		}
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestHandlerOfLeaderSynchronizesToFutureValidBlockWithModeUpdateOnly(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessWaitingForCommittedMessages(t, ctx)
		aBlockFromLeader := builders.BlockPair().WithBenchmarkConsensusBlockProof(leaderKeyPair())

		t.Log("Handle block consensus (ie due to block sync) of height 1002")

		b1002 := aBlockFromLeader.WithHeight(1002).WithInvalidBenchmarkConsensusBlockProof(leaderKeyPair()).Build()
		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(1002, h.config.NodeAddress())

		err := h.handleBlockConsensus(ctx, handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY, b1002, nil)
		if err != nil {
			t.Fatal("handle did not validate valid block:", err)
		}
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestHandlerOfLeaderIgnoresFutureValidBlockWithModeVerifyOnly(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessWaitingForCommittedMessages(t, ctx)
		aBlockFromLeader := builders.BlockPair().WithBenchmarkConsensusBlockProof(leaderKeyPair())

		t.Log("Handle block consensus (ie due to block sync) of height 1002")

		b1001 := aBlockFromLeader.WithHeight(1001).Build()
		b1002 := aBlockFromLeader.WithHeight(1002).WithPrevBlock(b1001).Build()
		h.expectNewBlockProposalNotRequested()
		h.expectCommitNotSent()

		err := h.handleBlockConsensus(ctx, handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_ONLY, b1002, b1001)
		if err != nil {
			t.Fatal("handle did not validate valid block:", err)
		}
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitNotSent(t)
	})
}

func TestHandlerOfNonLeaderSynchronizesToFutureValidBlock(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newNonLeaderHarness(t, ctx)
		aBlockFromLeader := builders.BlockPair().WithBenchmarkConsensusBlockProof(leaderKeyPair())

		t.Log("Handle block consensus (ie due to block sync) of height 1002")

		b1001 := aBlockFromLeader.WithHeight(1001).Build()
		b1002 := aBlockFromLeader.WithHeight(1002).WithPrevBlock(b1001).Build()

		err := h.handleBlockConsensus(ctx, handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE, b1002, b1001)
		if err != nil {
			t.Fatal("handle did not validate valid block:", err)
		}

		t.Log("Leader commits height 1003, confirm height 1003")

		b1003 := aBlockFromLeader.WithHeight(1003).WithPrevBlock(b1002).Build()
		h.expectCommitSaveAndReply(b1003, 1003, h.config.BenchmarkConsensusConstantLeader(), h.config.NodeAddress())

		h.receivedCommitViaGossip(ctx, b1003)
		h.verifyCommitSaveAndReply(t)
	})
}

func TestHandlerForBlockConsensusWithBadPrevBlockHashPointer(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newNonLeaderHarness(t, ctx)
		aBlockFromLeader := builders.BlockPair().WithBenchmarkConsensusBlockProof(leaderKeyPair())

		t.Log("Handle block consensus (ie due to block sync) of height 2 without hash pointer")

		b1 := aBlockFromLeader.WithHeight(1).Build()
		b2 := aBlockFromLeader.WithHeight(2).WithPrevBlock(nil).Build()

		err := h.handleBlockConsensus(ctx, handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE, b2, b1)
		if err == nil {
			t.Fatal("handle did not discover blocks with bad hash pointers:", err)
		}
	})
}

func TestHandlerForBlockConsensusWithBadSignature(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newNonLeaderHarness(t, ctx)
		aBlockFromLeader := builders.BlockPair().WithBenchmarkConsensusBlockProof(leaderKeyPair())

		t.Log("Handle block consensus (ie due to block sync) of height 2 with bad signature")

		b1 := aBlockFromLeader.WithHeight(1).Build()
		b2 := builders.BlockPair().
			WithHeight(2).
			WithPrevBlock(b1).
			WithInvalidBenchmarkConsensusBlockProof(leaderKeyPair()).
			Build()

		err := h.handleBlockConsensus(ctx, handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE, b2, b1)
		if err == nil {
			t.Fatal("handle did not discover blocks with bad signature:", err)
		}
	})
}

func TestHandlerForBlockConsensusFromNonLeader(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newNonLeaderHarness(t, ctx)
		aBlockFromNonLeader := builders.BlockPair().WithBenchmarkConsensusBlockProof(otherNonLeaderKeyPair())

		t.Log("Handle block consensus (ie due to block sync) of height 2 from non leader")

		b1 := aBlockFromNonLeader.WithHeight(1).Build()
		b2 := aBlockFromNonLeader.WithHeight(2).WithPrevBlock(b1).Build()

		err := h.handleBlockConsensus(ctx, handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE, b2, b1)
		if err == nil {
			t.Fatal("handle did not discover blocks not from the leader:", err)
		}
	})
}

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
	"testing"
)

func newNonLeaderHarness(t *testing.T, ctx context.Context) *harness {
	h := newHarness(t, false)
	h.createService(ctx)
	return h
}

func TestNonLeaderInit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newNonLeaderHarness(t, ctx)

		h.verifyHandlerRegistrations(t)
	})
}

func TestNonLeaderDoesNotProposeBlocks(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t, false)
		h.expectNewBlockProposalNotRequested()

		h.createService(ctx)
		h.verifyNewBlockProposalNotRequested(t)
	})
}

func TestNonLeaderRepliesToGenesisBlockCommit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newNonLeaderHarness(t, ctx)
		aBlockFromLeader := builders.BlockPair().WithBenchmarkConsensusBlockProof(leaderKeyPair())

		t.Log("Leader commits height 0 (genesis), confirm height 0")

		b0 := aBlockFromLeader.WithHeight(0).Build()
		h.expectCommitReplyWithoutSave(b0, 0, h.config.BenchmarkConsensusConstantLeader(), h.config.NodeAddress())

		h.receivedCommitViaGossip(ctx, b0)
		h.verifyCommitReplyWithoutSave(t)
	})
}

func TestNonLeaderSavesAndRepliesToConsecutiveBlockCommits(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newNonLeaderHarness(t, ctx)
		aBlockFromLeader := builders.BlockPair().WithBenchmarkConsensusBlockProof(leaderKeyPair())

		t.Log("Leader commits height 1, confirm height 1")

		b1 := aBlockFromLeader.WithHeight(1).Build()
		h.expectCommitSaveAndReply(b1, 1, h.config.BenchmarkConsensusConstantLeader(), h.config.NodeAddress())

		h.receivedCommitViaGossip(ctx, b1)
		h.verifyCommitSaveAndReply(t)

		t.Log("Leader commits height 2, confirm height 2")

		b2 := aBlockFromLeader.WithHeight(2).WithPrevBlock(b1).Build()
		h.expectCommitSaveAndReply(b2, 2, h.config.BenchmarkConsensusConstantLeader(), h.config.NodeAddress())

		h.receivedCommitViaGossip(ctx, b2)
		h.verifyCommitSaveAndReply(t)
	})
}

func TestNonLeaderSavesAndRepliesToAnOldBlockCommit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newNonLeaderHarness(t, ctx)
		aBlockFromLeader := builders.BlockPair().WithBenchmarkConsensusBlockProof(leaderKeyPair())

		t.Log("Leader commits height 1, confirm height 1")

		b1 := aBlockFromLeader.WithHeight(1).Build()
		h.expectCommitSaveAndReply(b1, 1, h.config.BenchmarkConsensusConstantLeader(), h.config.NodeAddress())

		h.receivedCommitViaGossip(ctx, b1)
		h.verifyCommitSaveAndReply(t)

		t.Log("Leader commits height 2, confirm height 2")

		b2 := aBlockFromLeader.WithHeight(2).WithPrevBlock(b1).Build()
		h.expectCommitSaveAndReply(b2, 2, h.config.BenchmarkConsensusConstantLeader(), h.config.NodeAddress())

		h.receivedCommitViaGossip(ctx, b2)
		h.verifyCommitSaveAndReply(t)

		t.Log("Leader commits height 1 again, confirm height 2 again")

		h.expectCommitSaveAndReply(b1, 2, h.config.BenchmarkConsensusConstantLeader(), h.config.NodeAddress())

		h.receivedCommitViaGossip(ctx, b1)
		h.verifyCommitSaveAndReply(t)
	})
}

func TestNonLeaderIgnoresFutureBlockCommit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newNonLeaderHarness(t, ctx)
		aBlockFromLeader := builders.BlockPair().WithBenchmarkConsensusBlockProof(leaderKeyPair())

		t.Log("Leader commits height 1000, don't confirm")

		b1000 := aBlockFromLeader.WithHeight(1000).Build()
		h.expectCommitIgnored()

		h.receivedCommitViaGossip(ctx, b1000)
		h.verifyCommitIgnored(t)
	})
}

func TestNonLeaderIgnoresBadPrevBlockHashPointer(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newNonLeaderHarness(t, ctx)
		aBlockFromLeader := builders.BlockPair().WithBenchmarkConsensusBlockProof(leaderKeyPair())

		t.Log("Leader commits height 1, confirm height 1")

		b1 := aBlockFromLeader.WithHeight(1).Build()
		h.expectCommitSaveAndReply(b1, 1, h.config.BenchmarkConsensusConstantLeader(), h.config.NodeAddress())

		h.receivedCommitViaGossip(ctx, b1)
		h.verifyCommitSaveAndReply(t)

		t.Log("Leader commits height 2 without hash pointer, don't confirm")

		b2 := aBlockFromLeader.WithHeight(2).WithPrevBlock(nil).Build()
		h.expectCommitIgnored()

		h.receivedCommitViaGossip(ctx, b2)
		h.verifyCommitIgnored(t)
	})
}

func TestNonLeaderIgnoresBadSignature(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newNonLeaderHarness(t, ctx)

		t.Log("Leader commits height 1 with bad signature, don't confirm")

		b1 := builders.BlockPair().
			WithHeight(1).
			WithInvalidBenchmarkConsensusBlockProof(leaderKeyPair()).
			Build()
		h.expectCommitIgnored()

		h.receivedCommitViaGossip(ctx, b1)
		h.verifyCommitIgnored(t)
	})
}

func TestNonLeaderIgnoresBlocksFromNonLeader(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newNonLeaderHarness(t, ctx)

		aBlockFromNonLeader := builders.BlockPair().WithBenchmarkConsensusBlockProof(otherNonLeaderKeyPair())

		t.Log("Non leader commits height 1, don't confirm")

		b1 := aBlockFromNonLeader.WithHeight(1).Build()
		h.expectCommitIgnored()

		h.receivedCommitViaGossip(ctx, b1)
		h.verifyCommitIgnored(t)
	})
}

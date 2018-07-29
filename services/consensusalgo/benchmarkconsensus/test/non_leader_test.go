package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"testing"
)

var leaderPublicKey, leaderPrivateKey = leaderKeyPair()

func newNonLeaderHarnessAndInit(t *testing.T, ctx context.Context) *harness {
	h := newHarness(false)
	h.createService(ctx)
	return h
}

func TestNonLeaderInit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newNonLeaderHarnessAndInit(t, ctx)
		h.verifyHandlerRegistrations(t)
	})
}

func TestNonLeaderDoesNotProposeBlocks(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(false)
		h.expectNewBlockProposalNotRequested()
		h.createService(ctx)
		h.verifyNewBlockProposalNotRequested(t)
	})
}

func TestNonLeaderRepliesToGenesisBlockCommit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newNonLeaderHarnessAndInit(t, ctx)
		aBlockFromLeader := builders.BlockPair().WithBenchmarkConsensusBlockProof(leaderPublicKey, leaderPrivateKey)

		b0 := aBlockFromLeader.WithHeight(0).Build()
		h.expectCommitReplyWithoutSave(b0, 0, h.config.ConstantConsensusLeader(), h.config.NodePublicKey())
		h.receivedCommitViaGossip(b0)
		h.verifyCommitReplyWithoutSave(t)
	})
}

func TestNonLeaderSavesAndRepliesToConsecutiveBlockCommits(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newNonLeaderHarnessAndInit(t, ctx)
		aBlockFromLeader := builders.BlockPair().WithBenchmarkConsensusBlockProof(leaderPublicKey, leaderPrivateKey)

		b1 := aBlockFromLeader.WithHeight(1).Build()
		h.expectCommitSaveAndReply(b1, 1, h.config.ConstantConsensusLeader(), h.config.NodePublicKey())
		h.receivedCommitViaGossip(b1)
		h.verifyCommitSaveAndReply(t)

		b2 := aBlockFromLeader.WithHeight(2).WithPrevBlockHash(b1).Build()
		h.expectCommitSaveAndReply(b2, 2, h.config.ConstantConsensusLeader(), h.config.NodePublicKey())
		h.receivedCommitViaGossip(b2)
		h.verifyCommitSaveAndReply(t)

		b3 := aBlockFromLeader.WithHeight(3).WithPrevBlockHash(b2).Build()
		h.expectCommitSaveAndReply(b3, 3, h.config.ConstantConsensusLeader(), h.config.NodePublicKey())
		h.receivedCommitViaGossip(b3)
		h.verifyCommitSaveAndReply(t)
	})
}

func TestNonLeaderSavesAndRepliesToAnOldBlockCommit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newNonLeaderHarnessAndInit(t, ctx)
		aBlockFromLeader := builders.BlockPair().WithBenchmarkConsensusBlockProof(leaderPublicKey, leaderPrivateKey)

		b1 := aBlockFromLeader.WithHeight(1).Build()
		h.expectCommitSaveAndReply(b1, 1, h.config.ConstantConsensusLeader(), h.config.NodePublicKey())
		h.receivedCommitViaGossip(b1)
		h.verifyCommitSaveAndReply(t)

		b2 := aBlockFromLeader.WithHeight(2).WithPrevBlockHash(b1).Build()
		h.expectCommitSaveAndReply(b2, 2, h.config.ConstantConsensusLeader(), h.config.NodePublicKey())
		h.receivedCommitViaGossip(b2)
		h.verifyCommitSaveAndReply(t)

		// sending b1 again (an old valid block)
		h.expectCommitSaveAndReply(b1, 2, h.config.ConstantConsensusLeader(), h.config.NodePublicKey())
		h.receivedCommitViaGossip(b1)
		h.verifyCommitSaveAndReply(t)
	})
}

func TestNonLeaderIgnoresFutureBlockCommit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newNonLeaderHarnessAndInit(t, ctx)
		aBlockFromLeader := builders.BlockPair().WithBenchmarkConsensusBlockProof(leaderPublicKey, leaderPrivateKey)

		b1000 := aBlockFromLeader.WithHeight(1000).Build()
		h.expectCommitIgnored()
		h.receivedCommitViaGossip(b1000)
		h.verifyCommitIgnored(t)
	})
}

func TestNonLeaderIgnoresBadPrevBlockHashPointer(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newNonLeaderHarnessAndInit(t, ctx)
		aBlockFromLeader := builders.BlockPair().WithBenchmarkConsensusBlockProof(leaderPublicKey, leaderPrivateKey)

		b1 := aBlockFromLeader.WithHeight(1).Build()
		h.expectCommitSaveAndReply(b1, 1, h.config.ConstantConsensusLeader(), h.config.NodePublicKey())
		h.receivedCommitViaGossip(b1)
		h.verifyCommitSaveAndReply(t)

		b2 := aBlockFromLeader.WithHeight(2).Build()
		h.expectCommitIgnored()
		h.receivedCommitViaGossip(b2)
		h.verifyCommitIgnored(t)
	})
}

func TestNonLeaderIgnoresBadSignature(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newNonLeaderHarnessAndInit(t, ctx)

		b1 := builders.BlockPair().
			WithHeight(1).
			WithInvalidBenchmarkConsensusBlockProof(leaderPublicKey, leaderPrivateKey).
			Build()
		h.expectCommitIgnored()
		h.receivedCommitViaGossip(b1)
		h.verifyCommitIgnored(t)
	})
}

func TestNonLeaderIgnoresBlocksFromNonLeader(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newNonLeaderHarnessAndInit(t, ctx)

		otherNonLeaderPublicKey, otherNonLeaderPrivateKey := otherNonLeaderKeyPair()
		aBlockFromNonLeader := builders.BlockPair().WithBenchmarkConsensusBlockProof(otherNonLeaderPublicKey, otherNonLeaderPrivateKey)

		b1 := aBlockFromNonLeader.WithHeight(1).Build()
		h.expectCommitIgnored()
		h.receivedCommitViaGossip(b1)
		h.verifyCommitIgnored(t)
	})
}

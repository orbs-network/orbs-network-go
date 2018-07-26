package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"testing"
)

var privateKey = []byte{147, 233, 25, 152, 106, 34, 71, 127, 218, 1, 103, 137, 204, 163, 12, 184, 65, 161, 53, 101, 9, 56, 113, 79, 133, 240, 0, 10, 101, 7, 107, 212, 223, 192, 108, 91, 226, 74, 103, 173, 238, 128, 179, 90, 180, 241, 71, 187, 26, 53, 197, 95, 248, 94, 218, 105, 244, 14, 248, 39, 189, 222, 193, 115}

func TestNonLeaderDoesNotProposeBlocks(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(false)
		h.expectNewBlockProposalNotRequested()
		h.createService(ctx)
		h.verifyNewBlockProposalNotRequested(t)
	})
}

func TestNonLeaderSavesAndRepliesToConsecutiveBlockCommits(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(false)
		h.createService(ctx)

		aBlock := builders.BlockPair().WithBenchmarkConsensusBlockProof(privateKey, h.config.ConstantConsensusLeader())

		b1 := aBlock.WithHeight(1).Build()
		h.expectCommitSaveAndReply(b1, 1)
		h.receivedCommitViaGossip(b1)
		h.verifyCommitSaveAndReply(t)

		b2 := aBlock.WithHeight(2).WithPrevBlockHash(b1).Build()
		h.expectCommitSaveAndReply(b2, 2)
		h.receivedCommitViaGossip(b2)
		h.verifyCommitSaveAndReply(t)

		b3 := aBlock.WithHeight(3).WithPrevBlockHash(b2).Build()
		h.expectCommitSaveAndReply(b3, 3)
		h.receivedCommitViaGossip(b3)
		h.verifyCommitSaveAndReply(t)
	})
}

func TestNonLeaderSavesAndRepliesToAnOldBlockCommit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(false)
		h.createService(ctx)

		aBlock := builders.BlockPair().WithBenchmarkConsensusBlockProof(privateKey, h.config.ConstantConsensusLeader())

		b1 := aBlock.WithHeight(1).Build()
		h.expectCommitSaveAndReply(b1, 1)
		h.receivedCommitViaGossip(b1)
		h.verifyCommitSaveAndReply(t)

		b2 := aBlock.WithHeight(2).WithPrevBlockHash(b1).Build()
		h.expectCommitSaveAndReply(b2, 2)
		h.receivedCommitViaGossip(b2)
		h.verifyCommitSaveAndReply(t)

		// sending b1 again (an old valid block)
		h.expectCommitSaveAndReply(b1, 2)
		h.receivedCommitViaGossip(b1)
		h.verifyCommitSaveAndReply(t)
	})
}

func TestNonLeaderIgnoresFutureBlockCommit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(false)
		h.createService(ctx)

		aBlock := builders.BlockPair().WithBenchmarkConsensusBlockProof(privateKey, h.config.ConstantConsensusLeader())

		h.expectCommitIgnored()
		b1 := aBlock.WithHeight(1000).Build()
		h.receivedCommitViaGossip(b1)
		h.verifyCommitIgnored(t)
	})
}

func TestNonLeaderIgnoresBadPrevBlockHashPointer(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(false)
		h.createService(ctx)

		aBlock := builders.BlockPair().WithBenchmarkConsensusBlockProof(privateKey, h.config.ConstantConsensusLeader())

		b1 := aBlock.WithHeight(1).Build()
		h.expectCommitSaveAndReply(b1, 1)
		h.receivedCommitViaGossip(b1)
		h.verifyCommitSaveAndReply(t)

		b2 := aBlock.WithHeight(2).Build()
		h.expectCommitIgnored()
		h.receivedCommitViaGossip(b2)
		h.verifyCommitIgnored(t)
	})
}

func TestNonLeaderIgnoresBadSignature(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(false)
		h.createService(ctx)

		aBlock := builders.BlockPair().WithInvalidBenchmarkConsensusBlockProof(privateKey, h.config.ConstantConsensusLeader())

		b1 := aBlock.WithHeight(1).Build()
		h.expectCommitIgnored()
		h.receivedCommitViaGossip(b1)
		h.verifyCommitIgnored(t)
	})
}

func TestNonLeaderIgnoresBlocksFromNonLeader(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(false)
		h.createService(ctx)

		aBlock := builders.BlockPair().WithBenchmarkConsensusBlockProof(privateKey, nonLeaderPublicKey())

		b1 := aBlock.WithHeight(1).Build()
		h.expectCommitIgnored()
		h.receivedCommitViaGossip(b1)
		h.verifyCommitIgnored(t)
	})
}

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"testing"
)

func TestNonLeaderDoesNotCreateBlocks(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(false)
		h.expectNoBlockCreation()
		h.createService(ctx)
		h.verifyNoBlockCreation(t)
	})
}

func TestNonLeaderCommitsAndRepliesToValidBlockHeights(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(false)
		h.createService(ctx)

		var replyBlockHeight primitives.BlockHeight

		h.expectCommitSaveAndReply(&replyBlockHeight)
		b1 := builders.BlockPair().
			WithHeight(1).
			WithBenchmarkConsensusBlockProof(nil, h.config.ConstantConsensusLeader()). //TODO: fix private key
			Build()
		h.receiveCommit(b1)
		if replyBlockHeight != 1 {
			t.Fatalf("Replied committed with wrong last block height: %d", replyBlockHeight)
		}
		h.verifyCommitSaveAndReply(t)

		h.expectCommitSaveAndReply(&replyBlockHeight)
		b2 := builders.BlockPair().
			WithHeight(2).
			WithPrevBlockHash(b1).
			WithBenchmarkConsensusBlockProof(nil, h.config.ConstantConsensusLeader()).
			Build()
		h.receiveCommit(b2)
		if replyBlockHeight != 2 {
			t.Fatalf("Replied committed with wrong last block height: %d", replyBlockHeight)
		}
		h.verifyCommitSaveAndReply(t)

		h.expectCommitSaveAndReply(&replyBlockHeight)
		h.receiveCommit(b1)
		if replyBlockHeight != 2 {
			t.Fatalf("Replied committed with wrong last block height: %d", replyBlockHeight)
		}
		h.verifyCommitSaveAndReply(t)

		h.expectCommitSaveAndReply(&replyBlockHeight)
		b3 := builders.BlockPair().
			WithHeight(3).
			WithPrevBlockHash(b2).
			WithBenchmarkConsensusBlockProof(nil, h.config.ConstantConsensusLeader()).
			Build()
		h.receiveCommit(b3)
		if replyBlockHeight != 3 {
			t.Fatalf("Replied committed with wrong last block height: %d", replyBlockHeight)
		}
		h.verifyCommitSaveAndReply(t)
	})
}

func TestNonLeaderIgnoresFutureBlockHeight(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(false)
		h.createService(ctx)

		h.expectIgnoreCommit()
		b1 := builders.BlockPair().
			WithHeight(1000).
			WithBenchmarkConsensusBlockProof(nil, h.config.ConstantConsensusLeader()).
			Build()
		h.receiveCommit(b1)
		h.verifyIgnoreCommit(t)
	})
}

func TestNonLeaderIgnoresBadPrevBlockHashPointer(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(false)
		h.createService(ctx)

		h.expectCommitSaveAndReply(nil)
		b1 := builders.BlockPair().
			WithHeight(1).
			WithBenchmarkConsensusBlockProof(nil, h.config.ConstantConsensusLeader()).
			Build()
		h.receiveCommit(b1)
		h.verifyCommitSaveAndReply(t)

		h.expectIgnoreCommit()
		b2 := builders.BlockPair().
			WithHeight(2).
			WithBenchmarkConsensusBlockProof(nil, h.config.ConstantConsensusLeader()).
			Build()
		h.receiveCommit(b2)
		h.verifyIgnoreCommit(t)
	})
}

func TestNonLeaderIgnoresBadSignature(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(false)
		h.createService(ctx)

		h.expectIgnoreCommit()
		b1 := builders.BlockPair().
			WithHeight(1).
			Build()
		h.receiveCommit(b1)
		h.verifyIgnoreCommit(t)
	})
}

func TestNonLeaderIgnoresBlocksFromNonLeader(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(false)
		h.createService(ctx)

		h.expectIgnoreCommit()
		b1 := builders.BlockPair().
			WithHeight(1).
			WithBenchmarkConsensusBlockProof(nil, nonLeaderPublicKey()).
			Build()
		h.receiveCommit(b1)
		h.verifyIgnoreCommit(t)
	})
}

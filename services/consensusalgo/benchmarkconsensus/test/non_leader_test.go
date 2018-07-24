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

func TestNonLeaderIgnoresFutureBlockHeight(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(false)
		h.createService(ctx)

		h.expectIgnoreCommit()
		h.receiveCommit(builders.BlockPair().WithHeight(1000).Build())
		h.verifyIgnoreCommit(t)
	})
}

func TestNonLeaderCommitsAndRepliesToValidBlockHeights(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(false)
		h.createService(ctx)

		var replyBlockHeight primitives.BlockHeight

		h.expectCommitSaveAndReply(&replyBlockHeight)
		h.receiveCommit(builders.BlockPair().WithHeight(1).Build())
		if replyBlockHeight != 1 {
			t.Fatalf("Replied committed with wrong last block height: %d", replyBlockHeight)
		}
		h.verifyCommitSaveAndReply(t)

		h.expectCommitSaveAndReply(&replyBlockHeight)
		h.receiveCommit(builders.BlockPair().WithHeight(2).Build())
		if replyBlockHeight != 2 {
			t.Fatalf("Replied committed with wrong last block height: %d", replyBlockHeight)
		}
		h.verifyCommitSaveAndReply(t)

		h.expectCommitSaveAndReply(&replyBlockHeight)
		h.receiveCommit(builders.BlockPair().WithHeight(1).Build())
		if replyBlockHeight != 2 {
			t.Fatalf("Replied committed with wrong last block height: %d", replyBlockHeight)
		}
		h.verifyCommitSaveAndReply(t)

		h.expectCommitSaveAndReply(&replyBlockHeight)
		h.receiveCommit(builders.BlockPair().WithHeight(3).Build())
		if replyBlockHeight != 3 {
			t.Fatalf("Replied committed with wrong last block height: %d", replyBlockHeight)
		}
		h.verifyCommitSaveAndReply(t)
	})
}

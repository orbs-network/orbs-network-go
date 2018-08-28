package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"testing"
)

func newLeaderHarnessWaitingForCommittedMessages(t *testing.T, ctx context.Context, numPersistentBlocks primitives.BlockHeight) *harness {
	h := newHarness(true)
	h.expectLastPersistentBlockToBeQueriedInStorage(numPersistentBlocks)
	h.expectNewBlockProposalNotRequested()
	h.expectCommitBroadcastViaGossip(numPersistentBlocks, h.config.NodePublicKey())
	h.createService(ctx)
	h.verifyLastPersistentBlockToBeQueriedInStorage(t)
	h.verifyCommitBroadcastViaGossip(t)
	return h
}

func TestLeaderInit(t *testing.T) {
	t.Parallel()
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessWaitingForCommittedMessages(t, ctx, 0)

		h.verifyHandlerRegistrations(t)
		h.verifyNewBlockProposalNotRequested(t)
	})
}

func TestLeaderCommitsConsecutiveBlocksAfterEnoughConfirmations(t *testing.T) {
	t.Parallel()
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessWaitingForCommittedMessages(t, ctx, 0)

		t.Log("Nodes confirmed height 0 (genesis), commit height 1")

		c0 := multipleCommittedMessages().WithHeight(0).WithCountAboveQuorum().Build()
		h.expectNewBlockProposalRequestedAndSaved(1)
		h.expectCommitBroadcastViaGossip(1, h.config.NodePublicKey())

		h.receivedCommittedMessagesViaGossip(c0)
		h.verifyNewBlockProposalRequestedAndSaved(t)
		h.verifyCommitBroadcastViaGossip(t)

		t.Log("Nodes confirmed height 1, commit height 2")

		c1 := multipleCommittedMessages().WithHeight(1).WithCountAboveQuorum().Build()
		h.expectNewBlockProposalRequestedAndSaved(2)
		h.expectCommitBroadcastViaGossip(2, h.config.NodePublicKey())

		h.receivedCommittedMessagesViaGossip(c1)
		h.verifyNewBlockProposalRequestedAndSaved(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderRetriesCommitOnErrorGeneratingBlock(t *testing.T) {
	t.Parallel()
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessWaitingForCommittedMessages(t, ctx, 0)

		t.Log("Nodes confirmed height 0 (genesis), fail to generate height 1")

		c0 := multipleCommittedMessages().WithHeight(0).WithCountAboveQuorum().Build()
		h.expectNewBlockProposalRequestedToFail()
		h.expectCommitNotSent()

		h.receivedCommittedMessagesViaGossip(c0)
		h.verifyNewBlockProposalRequestedAndNotSaved(t)
		h.verifyCommitNotSent(t)

		t.Log("Stop failing to generate, commit height 1")

		h.expectNewBlockProposalRequestedAndSaved(1)
		h.expectCommitBroadcastViaGossip(1, h.config.NodePublicKey())

		h.verifyNewBlockProposalRequestedAndSaved(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderRetriesCommitAfterNotEnoughConfirmations(t *testing.T) {
	t.Parallel()
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessWaitingForCommittedMessages(t, ctx, 0)

		t.Log("Nodes confirmed height 0 (genesis), commit height 1")

		c0 := multipleCommittedMessages().WithHeight(0).WithCountAboveQuorum().Build()
		h.expectNewBlockProposalRequestedAndSaved(1)
		h.expectCommitBroadcastViaGossip(1, h.config.NodePublicKey())

		h.receivedCommittedMessagesViaGossip(c0)
		h.verifyNewBlockProposalRequestedAndSaved(t)
		h.verifyCommitBroadcastViaGossip(t)

		t.Log("Not enough nodes confirmed height 1, commit height 1 again")

		c1 := multipleCommittedMessages().WithHeight(1).WithCountBelowQuorum().Build()
		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(1, h.config.NodePublicKey())

		h.receivedCommittedMessagesViaGossip(c1)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderIgnoresBadCommittedMessageSignatures(t *testing.T) {
	t.Parallel()
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessWaitingForCommittedMessages(t, ctx, 0)

		t.Log("Bad signatures nodes confirmed height 0 (genesis), commit height 0 again")

		c0 := multipleCommittedMessages().WithHeight(0).WithInvalidSignatures().WithCountAboveQuorum().Build()
		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(0, h.config.NodePublicKey())

		h.receivedCommittedMessagesViaGossip(c0)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderIgnoresNonFederationSigners(t *testing.T) {
	t.Parallel()
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessWaitingForCommittedMessages(t, ctx, 0)

		t.Log("Non federation nodes confirmed height 0 (genesis), commit height 0 again")

		c0 := multipleCommittedMessages().WithHeight(0).FromNonFederationMembers().WithCountAboveQuorum().Build()
		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(0, h.config.NodePublicKey())

		h.receivedCommittedMessagesViaGossip(c0)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderIgnoresOldConfirmations(t *testing.T) {
	t.Parallel()
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessWaitingForCommittedMessages(t, ctx, 0)

		t.Log("Nodes confirmed height 0 (genesis), commit height 1")

		c0 := multipleCommittedMessages().WithHeight(0).WithCountAboveQuorum().Build()
		h.expectNewBlockProposalRequestedAndSaved(1)
		h.expectCommitBroadcastViaGossip(1, h.config.NodePublicKey())

		h.receivedCommittedMessagesViaGossip(c0)
		h.verifyNewBlockProposalRequestedAndSaved(t)
		h.verifyCommitBroadcastViaGossip(t)

		t.Log("Nodes confirmed height 0 (genesis) again, commit height 1 again")

		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(1, h.config.NodePublicKey())

		h.receivedCommittedMessagesViaGossip(c0)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderIgnoresFutureConfirmations(t *testing.T) {
	t.Parallel()
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessWaitingForCommittedMessages(t, ctx, 0)

		t.Log("Nodes confirmed height 1000, commit height 0 (genesis) again")

		c1000 := multipleCommittedMessages().WithHeight(1000).WithCountAboveQuorum().Build()
		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(0, h.config.NodePublicKey())

		h.receivedCommittedMessagesViaGossip(c1000)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

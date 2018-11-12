package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"testing"
)

func newLeaderHarnessWaitingForCommittedMessages(t *testing.T, ctx context.Context) *harness {
	h := newHarness(true)
	h.expectNewBlockProposalNotRequested()
	h.expectCommitBroadcastViaGossip(0, h.config.NodePublicKey())
	h.createService(ctx)
	h.verifyCommitBroadcastViaGossip(t)
	return h
}

func TestLeaderInit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessWaitingForCommittedMessages(t, ctx)

		h.verifyHandlerRegistrations(t)
		h.verifyNewBlockProposalNotRequested(t)
	})
}

func TestLeaderCommitsConsecutiveBlocksAfterEnoughConfirmations(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessWaitingForCommittedMessages(t, ctx)

		t.Log("Nodes confirmed height 0 (genesis), commit height 1")

		c0 := multipleCommittedMessages().WithHeight(0).WithCountAboveQuorum(h.config).Build()
		h.expectNewBlockProposalRequestedAndSaved(1)
		h.expectCommitBroadcastViaGossip(1, h.config.NodePublicKey())

		h.receivedCommittedMessagesViaGossip(ctx, c0)
		h.verifyNewBlockProposalRequestedAndSaved(t)
		h.verifyCommitBroadcastViaGossip(t)

		t.Log("Nodes confirmed height 1, commit height 2")

		c1 := multipleCommittedMessages().WithHeight(1).WithCountAboveQuorum(h.config).Build()
		h.expectNewBlockProposalRequestedAndSaved(2)
		h.expectCommitBroadcastViaGossip(2, h.config.NodePublicKey())

		h.receivedCommittedMessagesViaGossip(ctx, c1)
		h.verifyNewBlockProposalRequestedAndSaved(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderRetriesCommitOnErrorGeneratingBlock(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessWaitingForCommittedMessages(t, ctx)

		t.Log("Nodes confirmed height 0 (genesis), fail to generate height 1")

		c0 := multipleCommittedMessages().WithHeight(0).WithCountAboveQuorum(h.config).Build()
		h.expectNewBlockProposalRequestedToFail()
		h.expectCommitNotSent()

		h.receivedCommittedMessagesViaGossip(ctx, c0)
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
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessWaitingForCommittedMessages(t, ctx)

		t.Log("Nodes confirmed height 0 (genesis), commit height 1")

		c0 := multipleCommittedMessages().WithHeight(0).WithCountAboveQuorum(h.config).Build()
		h.expectNewBlockProposalRequestedAndSaved(1)
		h.expectCommitBroadcastViaGossip(1, h.config.NodePublicKey())

		h.receivedCommittedMessagesViaGossip(ctx, c0)
		h.verifyNewBlockProposalRequestedAndSaved(t)
		h.verifyCommitBroadcastViaGossip(t)

		t.Log("Not enough nodes confirmed height 1, commit height 1 again")

		c1 := multipleCommittedMessages().WithHeight(1).WithCountBelowQuorum(h.config).Build()
		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(1, h.config.NodePublicKey())

		h.receivedCommittedMessagesViaGossip(ctx, c1)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderIgnoresBadCommittedMessageSignatures(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessWaitingForCommittedMessages(t, ctx)

		t.Log("Bad signatures nodes confirmed height 0 (genesis), commit height 0 again")

		c0 := multipleCommittedMessages().WithHeight(0).WithInvalidSignatures().WithCountAboveQuorum(h.config).Build()
		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(0, h.config.NodePublicKey())

		h.receivedCommittedMessagesViaGossip(ctx, c0)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderIgnoresNonFederationSigners(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessWaitingForCommittedMessages(t, ctx)

		t.Log("Non federation nodes confirmed height 0 (genesis), commit height 0 again")

		c0 := multipleCommittedMessages().WithHeight(0).FromNonFederationMembers().WithCountAboveQuorum(h.config).Build()
		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(0, h.config.NodePublicKey())

		h.receivedCommittedMessagesViaGossip(ctx, c0)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderIgnoresOldConfirmations(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessWaitingForCommittedMessages(t, ctx)

		t.Log("Nodes confirmed height 0 (genesis), commit height 1")

		c0 := multipleCommittedMessages().WithHeight(0).WithCountAboveQuorum(h.config).Build()
		h.expectNewBlockProposalRequestedAndSaved(1)
		h.expectCommitBroadcastViaGossip(1, h.config.NodePublicKey())

		h.receivedCommittedMessagesViaGossip(ctx, c0)
		h.verifyNewBlockProposalRequestedAndSaved(t)
		h.verifyCommitBroadcastViaGossip(t)

		t.Log("Nodes confirmed height 0 (genesis) again, commit height 1 again")

		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(1, h.config.NodePublicKey())

		h.receivedCommittedMessagesViaGossip(ctx, c0)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderIgnoresFutureConfirmations(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessWaitingForCommittedMessages(t, ctx)

		t.Log("Nodes confirmed height 1000, commit height 0 (genesis) again")

		c1000 := multipleCommittedMessages().WithHeight(1000).WithCountAboveQuorum(h.config).Build()
		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(0, h.config.NodePublicKey())

		h.receivedCommittedMessagesViaGossip(ctx, c1000)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

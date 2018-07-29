package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"testing"
)

func newLeaderHarnessAndInit(t *testing.T, ctx context.Context) *harness {
	h := newHarness(true)
	h.expectNewBlockProposalNotRequested()
	h.expectCommitBroadcastViaGossip(0, h.config.NodePublicKey())
	h.createService(ctx)
	h.verifyCommitBroadcastViaGossip(t)
	return h
}

func TestLeaderInit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessAndInit(t, ctx)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyHandlerRegistrations(t)
	})
}

func TestLeaderCommitsNextBlockAfterEnoughConfirmations(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessAndInit(t, ctx)

		// TODO: fix spacing

		c0 := committedMessages().WithHeight(0).WithCountAboveQuorum().Build()
		h.expectNewBlockProposalRequestedAndSaved(1)
		h.expectCommitBroadcastViaGossip(1, h.config.NodePublicKey())
		h.receivedCommittedMessagesViaGossip(c0)
		h.verifyNewBlockProposalRequestedAndSaved(t)
		h.verifyCommitBroadcastViaGossip(t) // TODO: more descriptive names

		c1 := committedMessages().WithHeight(1).WithCountAboveQuorum().Build()
		h.expectNewBlockProposalRequestedAndSaved(2)
		h.expectCommitBroadcastViaGossip(2, h.config.NodePublicKey())
		h.receivedCommittedMessagesViaGossip(c1)
		h.verifyNewBlockProposalRequestedAndSaved(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderRetriesCommitOnError(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessAndInit(t, ctx)

		c0 := committedMessages().WithHeight(0).WithCountAboveQuorum().Build()
		h.expectNewBlockProposalRequestedToFail()
		h.expectCommitNotSent()
		h.receivedCommittedMessagesViaGossip(c0)
		h.verifyNewBlockProposalRequestedAndNotSaved(t)
		h.verifyCommitNotSent(t)

		// TODO: add the missing half of the test
	})
}

func TestLeaderRetriesCommitAfterNotEnoughConfirmations(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessAndInit(t, ctx)

		c0 := committedMessages().WithHeight(0).WithCountAboveQuorum().Build()
		h.expectNewBlockProposalRequestedAndSaved(1)
		h.expectCommitBroadcastViaGossip(1, h.config.NodePublicKey())
		h.receivedCommittedMessagesViaGossip(c0)
		h.verifyNewBlockProposalRequestedAndSaved(t)
		h.verifyCommitBroadcastViaGossip(t)

		c1 := committedMessages().WithHeight(1).WithCountBelowQuorum().Build()
		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(1, h.config.NodePublicKey())
		h.receivedCommittedMessagesViaGossip(c1)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderIgnoresBadSignatures(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessAndInit(t, ctx)

		c0 := committedMessages().WithHeight(0).WithInvalidSignatures().WithCountAboveQuorum().Build()
		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(0, h.config.NodePublicKey())
		h.receivedCommittedMessagesViaGossip(c0)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderIgnoresNonFederationSigners(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessAndInit(t, ctx)

		c0 := committedMessages().WithHeight(0).FromNonFederationMembers().WithCountAboveQuorum().Build()
		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(0, h.config.NodePublicKey())
		h.receivedCommittedMessagesViaGossip(c0)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderIgnoresOldConfirmations(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessAndInit(t, ctx)

		c0 := committedMessages().WithHeight(0).WithCountAboveQuorum().Build()
		h.expectNewBlockProposalRequestedAndSaved(1)
		h.expectCommitBroadcastViaGossip(1, h.config.NodePublicKey())
		h.receivedCommittedMessagesViaGossip(c0)
		h.verifyNewBlockProposalRequestedAndSaved(t)
		h.verifyCommitBroadcastViaGossip(t)

		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(1, h.config.NodePublicKey())
		h.receivedCommittedMessagesViaGossip(c0)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderIgnoresFutureConfirmations(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessAndInit(t, ctx)

		c1000 := committedMessages().WithHeight(1000).WithCountAboveQuorum().Build()
		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(0, h.config.NodePublicKey())
		h.receivedCommittedMessagesViaGossip(c1000)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

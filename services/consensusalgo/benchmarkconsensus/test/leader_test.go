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

		h.expectNewBlockProposalRequestedAndSaved(1)
		h.expectCommitBroadcastViaGossip(1, h.config.NodePublicKey())

		// TODO: change to a builder and put it in the first line
		h.receivedCommittedViaGossipFromSeveral(3, 0, true, true)

		h.verifyNewBlockProposalRequestedAndSaved(t)
		h.verifyCommitBroadcastViaGossip(t) // TODO: more descriptive names

		h.expectNewBlockProposalRequestedAndSaved(2)
		h.expectCommitBroadcastViaGossip(2, h.config.NodePublicKey())
		h.receivedCommittedViaGossipFromSeveral(3, 1, true, true)
		h.verifyNewBlockProposalRequestedAndSaved(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderRetriesCommitOnError(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessAndInit(t, ctx)

		h.expectNewBlockProposalRequestedToFail()
		h.expectCommitNotSent()
		h.receivedCommittedViaGossipFromSeveral(3, 0, true, true)
		h.verifyNewBlockProposalRequestedAndNotSaved(t)
		h.verifyCommitNotSent(t)

		// TODO: add the missing half of the test
	})
}

func TestLeaderRetriesCommitAfterNotEnoughConfirmations(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessAndInit(t, ctx)

		h.expectNewBlockProposalRequestedAndSaved(1)
		h.expectCommitBroadcastViaGossip(1, h.config.NodePublicKey())
		h.receivedCommittedViaGossipFromSeveral(3, 0, true, true)
		h.verifyNewBlockProposalRequestedAndSaved(t)
		h.verifyCommitBroadcastViaGossip(t)

		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(1, h.config.NodePublicKey())
		h.receivedCommittedViaGossipFromSeveral(2, 1, true, true)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderIgnoresBadSignatures(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessAndInit(t, ctx)

		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(0, h.config.NodePublicKey())
		h.receivedCommittedViaGossipFromSeveral(3, 0, false, true)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderIgnoresNonFederationSigners(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessAndInit(t, ctx)

		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(0, h.config.NodePublicKey())
		h.receivedCommittedViaGossipFromSeveral(3, 0, true, false)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderIgnoresOldConfirmations(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessAndInit(t, ctx)

		h.expectNewBlockProposalRequestedAndSaved(1)
		h.expectCommitBroadcastViaGossip(1, h.config.NodePublicKey())
		h.receivedCommittedViaGossipFromSeveral(3, 0, true, true)
		h.verifyNewBlockProposalRequestedAndSaved(t)
		h.verifyCommitBroadcastViaGossip(t)

		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(1, h.config.NodePublicKey())
		h.receivedCommittedViaGossipFromSeveral(3, 0, true, true)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

func TestLeaderIgnoresFutureConfirmations(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessAndInit(t, ctx)

		h.expectNewBlockProposalNotRequested()
		h.expectCommitBroadcastViaGossip(0, h.config.NodePublicKey())
		h.receivedCommittedViaGossipFromSeveral(3, 1000, true, true)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitBroadcastViaGossip(t)
	})
}

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"testing"
)

func newLeaderHarnessAndInit(t *testing.T, ctx context.Context) *harness {
	h := newHarness(true)
	h.expectNewBlockProposalNotRequested()
	h.expectCommitSent(0, h.config.NodePublicKey())
	h.createService(ctx)
	h.verifyCommitSent(t)
	return h
}

func TestLeaderInit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessAndInit(t, ctx)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyHandlerRegistrations(t)
	})
}

// TODO: check it's from the approved list
func TestLeaderCommitsNextBlockAfterEnoughConfirmations(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessAndInit(t, ctx)

		h.expectNewBlockProposalRequested(1)
		h.expectCommitSent(1, h.config.NodePublicKey())
		h.receivedCommittedViaGossipFromSeveral(3, 0, true)
		h.verifyNewBlockProposalRequested(t)
		h.verifyCommitSent(t)

		h.expectNewBlockProposalRequested(2)
		h.expectCommitSent(2, h.config.NodePublicKey())
		h.receivedCommittedViaGossipFromSeveral(3, 1, true)
		h.verifyNewBlockProposalRequested(t)
		h.verifyCommitSent(t)
	})
}

func TestLeaderRetriesCommitOnError(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessAndInit(t, ctx)

		h.expectNewBlockProposalRequestedToFail()
		h.expectCommitNotSent()
		h.receivedCommittedViaGossipFromSeveral(3, 0, true)
		h.verifyNewBlockProposalRequested(t)
		h.verifyCommitNotSent(t)
	})
}

func TestLeaderRetriesCommitAfterNotEnoughConfirmations(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessAndInit(t, ctx)

		h.expectNewBlockProposalRequested(1)
		h.expectCommitSent(1, h.config.NodePublicKey())
		h.receivedCommittedViaGossipFromSeveral(3, 0, true)
		h.verifyNewBlockProposalRequested(t)
		h.verifyCommitSent(t)

		h.expectNewBlockProposalNotRequested()
		h.expectCommitSent(1, h.config.NodePublicKey())
		h.receivedCommittedViaGossipFromSeveral(2, 1, true)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitSent(t)
	})
}

func TestLeaderRetriesCommitAfterEnoughBadSignatures(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessAndInit(t, ctx)

		h.expectNewBlockProposalNotRequested()
		h.expectCommitSent(0, h.config.NodePublicKey())
		h.receivedCommittedViaGossipFromSeveral(3, 0, false)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitSent(t)
	})
}

func TestLeaderIgnoresOldConfirmations(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessAndInit(t, ctx)

		h.expectNewBlockProposalRequested(1)
		h.expectCommitSent(1, h.config.NodePublicKey())
		h.receivedCommittedViaGossipFromSeveral(3, 0, true)
		h.verifyNewBlockProposalRequested(t)
		h.verifyCommitSent(t)

		h.expectNewBlockProposalNotRequested()
		h.expectCommitSent(1, h.config.NodePublicKey())
		h.receivedCommittedViaGossipFromSeveral(3, 0, true)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitSent(t)
	})
}

func TestLeaderIgnoresFutureConfirmations(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeaderHarnessAndInit(t, ctx)

		h.expectNewBlockProposalNotRequested()
		h.expectCommitSent(0, h.config.NodePublicKey())
		h.receivedCommittedViaGossipFromSeveral(3, 1000, true)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitSent(t)
	})
}

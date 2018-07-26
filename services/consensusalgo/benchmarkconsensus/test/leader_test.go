package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"testing"
)

func TestLeaderInit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(true)
		h.expectNewBlockProposalRequestedToFail()
		h.createService(ctx)
		h.verifyHandlerRegistrations(t)
	})
}

func TestLeaderCommitsValidFirstBlock(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(true)
		h.expectNewBlockProposalRequested(1)
		h.expectCommitSent(1, h.config.NodePublicKey())
		h.createService(ctx)
		h.verifyNewBlockProposalRequested(t)
		h.verifyCommitSent(t)
	})
}

func TestLeaderRetriesCommitOnError(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(true)
		h.expectNewBlockProposalRequestedToFail()
		h.expectCommitNotSent()
		h.createService(ctx)
		h.verifyNewBlockProposalRequested(t)
		h.verifyCommitNotSent(t)

		h.expectNewBlockProposalRequested(1)
		h.expectCommitSent(1, h.config.NodePublicKey())
		h.verifyNewBlockProposalRequested(t)
		h.verifyCommitSent(t)
	})
}

// TODO: check it's from the approved list
func TestLeaderCommitsSecondBlockAfterEnoughConfirmations(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(true)
		h.expectNewBlockProposalRequested(1)
		h.expectCommitSent(1, h.config.NodePublicKey())
		h.createService(ctx)
		h.verifyNewBlockProposalRequested(t)
		h.verifyCommitSent(t)

		h.expectNewBlockProposalRequested(2)
		h.expectCommitSent(2, h.config.NodePublicKey())
		h.receivedCommittedViaGossipFromSeveral(3, 1, true)
		h.verifyNewBlockProposalRequested(t)
		h.verifyCommitSent(t)
	})
}

func TestLeaderRetriesCommitAfterEnoughBadSignatures(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(true)
		h.expectNewBlockProposalRequested(1)
		h.expectCommitSent(1, h.config.NodePublicKey())
		h.createService(ctx)
		h.verifyNewBlockProposalRequested(t)
		h.verifyCommitSent(t)

		h.expectNewBlockProposalNotRequested()
		h.expectCommitSent(1, h.config.NodePublicKey())
		h.receivedCommittedViaGossipFromSeveral(3, 1, false)
		h.verifyNewBlockProposalNotRequested(t)
		h.verifyCommitSent(t)
	})
}

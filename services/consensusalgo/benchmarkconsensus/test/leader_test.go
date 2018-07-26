package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"testing"
)

func TestLeaderCommitsValidFirstBlock(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(true)
		h.expectNewBlockProposalRequested(1, nil)
		h.expectCommitSent(1, h.config.NodePublicKey())
		h.createService(ctx)
		h.verifyNewBlockProposalRequested(t)
		h.verifyCommitSent(t)
	})
}

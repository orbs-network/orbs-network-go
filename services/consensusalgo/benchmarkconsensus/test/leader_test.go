package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"testing"
)

func TestLeaderCreatesBlocks(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(true)
		h.expectNewBlockProposalRequested()
		h.createService(ctx)
		h.verifyNewBlockProposalRequested(t)
	})
}

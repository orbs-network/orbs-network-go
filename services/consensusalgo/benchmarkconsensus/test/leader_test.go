package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test"
	"testing"
)

func TestLeaderCreatesBlocks(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(true)
		h.consensusContext.Reset().When("RequestNewTransactionsBlock", mock.Any).Return(nil, nil).AtLeast(1)
		h.createService(ctx)
		err := test.EventuallyVerify(h.consensusContext)
		if err != nil {
			t.Fatal("Did not create block with ConsensusContext:", err)
		}
	})
}

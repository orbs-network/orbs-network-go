package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"testing"
)

func TestNonLeaderHandlesValidBlockConsensus(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newNonLeaderHarnessAndInit(t, ctx)

		b1 := aBlockFromLeader.WithHeight(1).Build()
		b2 := aBlockFromLeader.WithHeight(2).WithPrevBlockHash(b1).Build()
		err := h.handleBlockConsensus(b2, b1)
		if err != nil {
			t.Fatal("handle did not validate valid block:", err)
		}
	})
}

// TODO: rely on future block to set lastCommitted

package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test"
	"testing"
)

func TestInit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(true)
		h.gossip.Reset().When("RegisterBenchmarkConsensusHandler", mock.Any).Return().Times(1)
		h.blockStorage.Reset().When("RegisterConsensusBlocksHandler", mock.Any).Return().Times(1)
		h.createService(ctx)

		ok, err := h.gossip.Verify()
		if !ok {
			t.Fatal("Did not register with Gossip:", err)
		}
		ok, err = h.blockStorage.Verify()
		if !ok {
			t.Fatal("Did not register with BlockStorage:", err)
		}
	})
}

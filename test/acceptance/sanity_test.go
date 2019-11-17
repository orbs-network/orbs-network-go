package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"testing"
)

func TestSanity(t *testing.T) {
	t.Skip("on purpose - this test is here to help us migrate acceptance network to ConcurrencyHarness")
	NewHarness().
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS).
		Start(t, func(t testing.TB, ctx context.Context, network *Network) {
			t.Fatalf("Intentional error")
		})
}

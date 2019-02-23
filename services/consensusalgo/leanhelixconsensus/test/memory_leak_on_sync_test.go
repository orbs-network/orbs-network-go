package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/stretchr/testify/require"
	"runtime"
	"testing"
)

const MAX_LEAK_BYTES = 5 * 1024 * 1024

func TestService_MemoryLeakOnBlockSync(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newLeanHelixServiceHarness().start(t, ctx)

		t.Log("Block sync service to block 5")

		h.expectConsensusContextRequestOrderingCommittee(1) // we're index 0 (first time called)

		b5 := builders.BlockPair().WithHeight(5).WithEmptyLeanHelixBlockProof().Build()
		h.consensus.HandleBlockConsensus(ctx, &handlers.HandleBlockConsensusInput{
			Mode:                   handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY,
			BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
			BlockPair:              b5,
			PrevCommittedBlockPair: nil,
		})

		require.NoError(t, test.EventuallyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT, h.consensusContext))

		t.Log("Listen to gossip consensus messages for multiple future blocks (during sync)")

		memUsageBefore := getMemUsageBytes()
		for bh := 1000; bh < 1040; bh++ {
			h.incomingLargeConsensusMessageViaGossip(ctx, primitives.BlockHeight(bh))
			memUsageAfter := getMemUsageBytes()
			require.Truef(t, memUsageAfter < memUsageBefore+MAX_LEAK_BYTES, "memory should not increase dramatically, increased by %d bytes", memUsageAfter-memUsageBefore)
		}
	})
}

func (h *harness) incomingLargeConsensusMessageViaGossip(ctx context.Context, blockHeight primitives.BlockHeight) {
	c := generatePreprepareMessage(h.instanceId, uint64(blockHeight), 0, "abc")
	b := builders.BlockPair().WithHeight(blockHeight).WithTransactions(1000).WithEmptyLeanHelixBlockProof().Build()
	h.consensus.HandleLeanHelixMessage(ctx, &gossiptopics.LeanHelixInput{
		Message: &gossipmessages.LeanHelixMessage{
			Content:   c.Content,
			BlockPair: b,
		},
	})
}

func getMemUsageBytes() uint64 {
	runtime.GC()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}

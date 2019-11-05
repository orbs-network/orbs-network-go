package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

func TestMetricsAreUpdatedOnElectionTrigger(t *testing.T) {

	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		h := newLeanHelixServiceHarness(0, 10*time.Millisecond).start(parent, ctx)

		h.beLastInCommittee()
		h.expectGossipSendLeanHelixMessage()

		b5 := builders.BlockPair().WithHeight(5).WithEmptyLeanHelixBlockProof().Build()

		_, err := h.consensus.HandleBlockConsensus(ctx, &handlers.HandleBlockConsensusInput{
			Mode:                   handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY,
			BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
			BlockPair:              b5,
			PrevCommittedBlockPair: nil,
		})
		require.NoError(t, err, "expected HandleBlockConsensus to succeed")
		require.NoError(t, test.EventuallyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT, h.consensusContext))

		metrics := h.getMetrics()

		require.True(t, test.Eventually(1*time.Second, func() bool {
			return metrics.currentElectionCount.Value() > 0
		}), "expected currentElectionCount metric to update")

		require.True(t, test.Eventually(1*time.Second, func() bool {
			return metrics.currentLeaderMemberId.Value() != ""
		}), "expected currentLeaderMemberId metric to update")

		require.True(t, test.Eventually(1*time.Second, func() bool {
			matched, err := regexp.MatchString("samples=[1-9]", metrics.timeSinceLastElectionMillis.String())
			if err != nil {
				panic(err)
			}
			return matched
		}), "expected timeSinceLastElectionMillis metric to update")
	})

}

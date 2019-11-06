package test

import (
	"context"
	"github.com/orbs-network/lean-helix-go/services/blockproof"
	"github.com/orbs-network/lean-helix-go/services/interfaces"
	"github.com/orbs-network/lean-helix-go/services/randomseed"
	"github.com/orbs-network/lean-helix-go/spec/types/go/protocol"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/stretchr/testify/require"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestMetricsAreUpdatedOnElectionTrigger(t *testing.T) {

	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		h := newLeanHelixServiceHarness(0, 10*time.Millisecond).start(parent, ctx)

		h.beFirstInCommittee() // just so RequestOrderedCommittee will succeed
		h.expectGossipSendLeanHelixMessage()

		h.handleBlockSync(ctx, 1)

		// Election trigger should fire and metrics should be updated

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

func TestMetricsAreUpdatedOnCommit(t *testing.T) {

	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		h := newLeanHelixServiceHarness(0, time.Hour).start(parent, ctx)

		h.expectGossipSendLeanHelixMessage()
		h.expectValidateTransactionBlock()
		h.expectValidateResultsBlock()
		h.expectCommitBlock()

		h.dontBeFirstInCommitee()

		const syncedBlockHeight = 1
		h.handleBlockSync(ctx, syncedBlockHeight)

		orderedCommittee := h.requestOrderingCommittee(ctx)
		orderedComitteeNodeIndicies := keys.NodeAddressesForTestsToIndexes(orderedCommittee.NodeAddresses)

		leaderNodeIndex := orderedComitteeNodeIndicies[0] // leader for view 0 is always the first member in the ordered committee

		prevBlockProof := protocol.BlockProofReader(nil) // the previous block (from sync) had no proof
		randomSeed := randomseed.CalculateRandomSeed(prevBlockProof.RandomSeedSignature())

		const firstTermHeight = syncedBlockHeight + 1
		const view = 0
		blockPair := builders.BlockPair().WithHeight(firstTermHeight).WithTransactions(1).WithEmptyLeanHelixBlockProof().Build()

		h.handlePreprepareMessage(ctx, blockPair, firstTermHeight, view, leaderNodeIndex)

		commitMessages := []*interfaces.CommitMessage{}
		for i := 0; i < h.networkSize(); i++ {
			cmsg := h.handleCommitMessage(ctx, blockPair, firstTermHeight, view, randomSeed, i)
			commitMessages = append(commitMessages, cmsg)
		}

		metrics := h.getMetrics()

		// At this point the first block should be committed, lastCommitTime should update
		require.True(t, test.Eventually(1*time.Second, func() bool {
			return metrics.lastCommittedTime.Value() != 0
		}), "expected lastCommittedTime metric not to update on first commit")

		// timeSinceLastCommitMillis will NOT update because this is the first commit
		require.True(t, strings.Contains(metrics.timeSinceLastCommitMillis.String(), "samples=0"), "expected lastCommittedTime to not update on first commit")

		// Starting the second term

		// Use commits from previous term to calculate the new random seed
		proof := blockproof.GenerateLeanHelixBlockProof(h.keyManagerForNode(h.myNodeIndex()), commitMessages)
		prevBlockProof = protocol.BlockProofReader(proof.Raw())
		randomSeed = randomseed.CalculateRandomSeed(prevBlockProof.RandomSeedSignature())

		const secondTermHeight = firstTermHeight + 1
		secondBlockPair := builders.BlockPair().WithHeight(secondTermHeight).WithTransactions(1).WithEmptyLeanHelixBlockProof().Build()

		h.handlePreprepareMessage(ctx, secondBlockPair, secondTermHeight, view, leaderNodeIndex)

		for i := 0; i < h.networkSize(); i++ {
			h.handleCommitMessage(ctx, secondBlockPair, secondTermHeight, view, randomSeed, i)
		}

		// A second commit should take place now and this time timeSinceLastCommitMillis should update

		require.True(t, test.Eventually(1*time.Second, func() bool {
			matched, err := regexp.MatchString("samples=[1-9]", metrics.timeSinceLastCommitMillis.String())
			if err != nil {
				panic(err)
			}
			return matched
		}), "expected timeSinceLastCommitMillis metric to update")
	})

}

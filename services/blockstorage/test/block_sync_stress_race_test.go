package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/stretchr/testify/require"
	"math/rand"
	"testing"
	"time"
)

func TestSyncPetitioner_Stress_CommitsDuringSync(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness(t).
			withSyncNoCommitTimeout(10 * time.Millisecond).
			withSyncCollectResponsesTimeout(1 * time.Millisecond).
			withSyncCollectChunksTimeout(50 * time.Millisecond)

		const NUM_BLOCKS = 300
		done := false

		harness.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).Call(func(ctx context.Context, input *gossiptopics.BlockAvailabilityRequestInput) (*gossiptopics.EmptyOutput, error) {
			respondToBroadcastAvailabilityRequest(t, ctx, harness, input, NUM_BLOCKS, 7)
			return nil, nil
		})

		harness.gossip.When("SendBlockSyncRequest", mock.Any, mock.Any).Call(func(ctx context.Context, input *gossiptopics.BlockSyncRequestInput) (*gossiptopics.EmptyOutput, error) {
			if input.Message.SignedChunkRange.LastBlockHeight() >= NUM_BLOCKS {
				done = true
			}
			respondToBlockSyncRequestWithConcurrentCommit(ctx, harness, input, NUM_BLOCKS)
			return nil, nil
		})

		harness.consensus.Reset().When("HandleBlockConsensus", mock.Any, mock.Any).Call(func(ctx context.Context, input *handlers.HandleBlockConsensusInput) (*handlers.HandleBlockConsensusOutput, error) {
			if input.Mode == handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE && input.PrevCommittedBlockPair != nil {
				currHeight := input.BlockPair.TransactionsBlock.Header.BlockHeight()
				prevHeight := input.PrevCommittedBlockPair.TransactionsBlock.Header.BlockHeight()
				if currHeight != prevHeight+1 {
					done = true
					require.Failf(t, "HandleBlockConsensus given invalid args", "called with height %d and prev height %d", currHeight, prevHeight)
				}
			}
			return nil, nil
		})

		harness.start(ctx)

		passed := test.Eventually(10*time.Second, func() bool { // wait for sync flow to complete successfully:
			return done
		})
		require.True(t, passed, "timed out waiting for passing conditions")
	})
}

func respondToBlockSyncRequestWithConcurrentCommit(ctx context.Context, harness *harness, input *gossiptopics.BlockSyncRequestInput, availableBlocks int) {
	response := builders.BlockSyncResponseInput().
		WithFirstBlockHeight(input.Message.SignedChunkRange.FirstBlockHeight()).
		WithLastBlockHeight(input.Message.SignedChunkRange.LastBlockHeight()).
		WithLastCommittedBlockHeight(primitives.BlockHeight(availableBlocks)).
		WithSenderNodeAddress(input.RecipientNodeAddress).Build()

	go func() {
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Nanosecond)
		harness.blockStorage.HandleBlockSyncResponse(ctx, response)
	}()

	go func() {
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Nanosecond)
		harness.blockStorage.CommitBlock(ctx, &services.CommitBlockInput{
			BlockPair: response.Message.BlockPairs[0],
		})
		harness.blockStorage.CommitBlock(ctx, &services.CommitBlockInput{
			BlockPair: response.Message.BlockPairs[1],
		})
	}()
}

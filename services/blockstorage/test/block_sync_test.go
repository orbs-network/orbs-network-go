package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

// TODO(v1) move to unit tests
func TestSyncSource_IgnoresRangesOfBlockSyncRequestAccordingToLocalBatchSettings(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness(t).withSyncBroadcast(1).start(ctx)

		blocks := []*protocol.BlockPairContainer{
			builders.BlockPair().WithHeight(primitives.BlockHeight(1)).WithBlockCreated(time.Now()).Build(),
			builders.BlockPair().WithHeight(primitives.BlockHeight(2)).WithBlockCreated(time.Now()).Build(),
			builders.BlockPair().WithHeight(primitives.BlockHeight(3)).WithBlockCreated(time.Now()).Build(),
			builders.BlockPair().WithHeight(primitives.BlockHeight(4)).WithBlockCreated(time.Now()).Build(),
		}

		harness.commitBlock(ctx, blocks[0])
		harness.commitBlock(ctx, blocks[1])
		harness.commitBlock(ctx, blocks[2])
		harness.commitBlock(ctx, blocks[3])

		expectedBlocks := []*protocol.BlockPairContainer{blocks[1], blocks[2]}

		senderKeyPair := keys.EcdsaSecp256K1KeyPairForTests(9)
		input := builders.BlockSyncRequestInput().
			WithFirstBlockHeight(primitives.BlockHeight(2)).
			WithLastBlockHeight(primitives.BlockHeight(10002)).
			WithLastCommittedBlockHeight(primitives.BlockHeight(2)).
			WithSenderNodeAddress(senderKeyPair.NodeAddress()).Build()

		response := &gossiptopics.BlockSyncResponseInput{
			RecipientNodeAddress: senderKeyPair.NodeAddress(),
			Message: &gossipmessages.BlockSyncResponseMessage{
				Sender: (&gossipmessages.SenderSignatureBuilder{
					SenderNodeAddress: harness.config.NodeAddress(),
				}).Build(),
				SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
					BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
					FirstBlockHeight:         primitives.BlockHeight(2),
					LastBlockHeight:          primitives.BlockHeight(3),
					LastCommittedBlockHeight: primitives.BlockHeight(4),
				}).Build(),
				BlockPairs: expectedBlocks,
			},
		}

		harness.gossip.When("SendBlockSyncResponse", mock.Any, response).Return(nil, nil).Times(1)

		_, err := harness.blockStorage.HandleBlockSyncRequest(ctx, input)
		require.NoError(t, err)

		harness.verifyMocks(t, 4)
	})
}

func TestSyncPetitioner_BroadcastsBlockAvailabilityRequest(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness(t).withSyncNoCommitTimeout(3 * time.Millisecond)
		harness.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).Return(nil, nil).AtLeast(2)

		harness.start(ctx)

		harness.verifyMocks(t, 2)
	})
}

func TestSyncPetitioner_CompleteSyncFlow(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness(t).
			withSyncCollectResponsesTimeout(50 * time.Millisecond).
			withSyncCollectChunksTimeout(50 * time.Millisecond)

		const NUM_BLOCKS = 4

		var results struct {
			sync.Mutex
			blocksSentBySource                map[primitives.BlockHeight]bool
			blocksReceivedByConsensus         map[primitives.BlockHeight]bool
			didUpdateConsensusAboutHeightZero bool
		}
		results.blocksSentBySource = make(map[primitives.BlockHeight]bool)
		results.blocksReceivedByConsensus = make(map[primitives.BlockHeight]bool)

		harness.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).Call(func(ctx context.Context, input *gossiptopics.BlockAvailabilityRequestInput) (*gossiptopics.EmptyOutput, error) {
			firstBlockHeight := input.Message.SignedBatchRange.FirstBlockHeight()
			if firstBlockHeight > NUM_BLOCKS {
				return nil, nil
			}

			response1 := builders.BlockAvailabilityResponseInput().
				WithLastCommittedBlockHeight(primitives.BlockHeight(NUM_BLOCKS)).
				WithFirstBlockHeight(firstBlockHeight).
				WithLastBlockHeight(primitives.BlockHeight(NUM_BLOCKS)).
				WithSenderNodeAddress(keys.EcdsaSecp256K1KeyPairForTests(7).NodeAddress()).Build()
			go harness.blockStorage.HandleBlockAvailabilityResponse(ctx, response1)

			response2 := builders.BlockAvailabilityResponseInput().
				WithLastCommittedBlockHeight(primitives.BlockHeight(NUM_BLOCKS)).
				WithFirstBlockHeight(firstBlockHeight).
				WithLastBlockHeight(primitives.BlockHeight(NUM_BLOCKS)).
				WithSenderNodeAddress(keys.EcdsaSecp256K1KeyPairForTests(8).NodeAddress()).Build()
			go harness.blockStorage.HandleBlockAvailabilityResponse(ctx, response2)

			return nil, nil
		})

		harness.gossip.When("SendBlockSyncRequest", mock.Any, mock.Any).Call(func(ctx context.Context, input *gossiptopics.BlockSyncRequestInput) (*gossiptopics.EmptyOutput, error) {
			require.Contains(t, []primitives.NodeAddress{
				keys.EcdsaSecp256K1KeyPairForTests(7).NodeAddress(),
				keys.EcdsaSecp256K1KeyPairForTests(8).NodeAddress(),
			}, input.RecipientNodeAddress, "the nodes accessed must be 7 or 8")

			require.Condition(t, func() (success bool) {
				return input.Message.SignedChunkRange.FirstBlockHeight() >= 1 && input.Message.SignedChunkRange.FirstBlockHeight() <= NUM_BLOCKS
			}, "first requested block must be between 1 and total")

			require.Condition(t, func() (success bool) {
				return input.Message.SignedChunkRange.LastBlockHeight() >= input.Message.SignedChunkRange.FirstBlockHeight() && input.Message.SignedChunkRange.LastBlockHeight() <= NUM_BLOCKS
			}, "last requested block must be between first and total")

			results.Lock()
			defer results.Unlock()
			for i := input.Message.SignedChunkRange.FirstBlockHeight(); i <= input.Message.SignedChunkRange.LastBlockHeight(); i++ {
				results.blocksSentBySource[i] = true
			}

			response := builders.BlockSyncResponseInput().
				WithFirstBlockHeight(input.Message.SignedChunkRange.FirstBlockHeight()).
				WithLastBlockHeight(input.Message.SignedChunkRange.LastBlockHeight()).
				WithLastCommittedBlockHeight(primitives.BlockHeight(NUM_BLOCKS)).
				WithSenderNodeAddress(input.RecipientNodeAddress).Build()
			go harness.blockStorage.HandleBlockSyncResponse(ctx, response)

			return nil, nil
		})

		harness.consensus.Reset().When("HandleBlockConsensus", mock.Any, mock.Any).Call(func(ctx context.Context, input *handlers.HandleBlockConsensusInput) (*handlers.HandleBlockConsensusOutput, error) {
			require.Contains(t, []handlers.HandleBlockConsensusMode{
				handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY,
				handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE,
			}, input.Mode, "consensus updates must be update or update+verify")

			results.Lock()
			defer results.Unlock()
			switch input.Mode {
			case handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY:
				if input.BlockPair == nil {
					results.didUpdateConsensusAboutHeightZero = true
				}
			case handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE:
				require.Condition(t, func() (success bool) {
					return input.BlockPair.TransactionsBlock.Header.BlockHeight() >= 1 && input.BlockPair.TransactionsBlock.Header.BlockHeight() <= NUM_BLOCKS
				}, "validated block must be between 1 and total")
				results.blocksReceivedByConsensus[input.BlockPair.TransactionsBlock.Header.BlockHeight()] = true
			}

			return nil, nil
		})

		harness.start(ctx)

		passed := test.Eventually(2*time.Second, func() bool {
			results.Lock()
			defer results.Unlock()
			if !results.didUpdateConsensusAboutHeightZero {
				return false
			}
			for i := primitives.BlockHeight(1); i < primitives.BlockHeight(NUM_BLOCKS); i++ {
				if !results.blocksSentBySource[i] || !results.blocksReceivedByConsensus[i] {
					return false
				}
			}
			return true
		})
		require.Truef(t, passed, "timed out waiting for passing conditions: %+v", results)
	})
}

func TestSyncPetitioner_NeverStartsWhenBlocksAreCommitted(t *testing.T) {
	t.Skip("this test needs to move to CommitBlock unit test, as a 'CommitBlockUpdatesBlockSync'")
	// this test may still be flaky, it runs commits in a busy wait loop that should take longer than the timeout,
	// to make sure we stay at the same state logically.
	// system timing may cause it to flake, but at a very low probability now
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness(t).
			withSyncNoCommitTimeout(5 * time.Millisecond).
			withSyncBroadcast(1).
			withCommitStateDiff(10).
			start(ctx)

		// we do not assume anything about the implementation, commit a block/ms and see if the sync tries to broadcast
		latch := make(chan struct{})
		go func() {
			for i := 1; i < 11; i++ {
				blockCreated := time.Now()
				blockHeight := primitives.BlockHeight(i)

				_, err := harness.commitBlock(ctx, builders.BlockPair().WithHeight(blockHeight).WithBlockCreated(blockCreated).Build())

				require.NoError(t, err)

				time.Sleep(500 * time.Microsecond)
			}
			latch <- struct{}{}
		}()

		<-latch
		require.EqualValues(t, 10, harness.numOfWrittenBlocks())
		harness.verifyMocks(t, 1)
	})
}

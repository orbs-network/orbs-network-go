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
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// TODO(v1) move to unit tests
func TestSyncSource_IgnoresRangesOfBlockSyncRequestAccordingToLocalBatchSettings(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness().withSyncBroadcast(1).start(ctx)

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

		senderKeyPair := keys.Ed25519KeyPairForTests(9)
		input := builders.BlockSyncRequestInput().
			WithFirstBlockHeight(primitives.BlockHeight(2)).
			WithLastBlockHeight(primitives.BlockHeight(10002)).
			WithLastCommittedBlockHeight(primitives.BlockHeight(2)).
			WithSenderPublicKey(senderKeyPair.PublicKey()).Build()

		response := &gossiptopics.BlockSyncResponseInput{
			RecipientPublicKey: senderKeyPair.PublicKey(),
			Message: &gossipmessages.BlockSyncResponseMessage{
				Sender: (&gossipmessages.SenderSignatureBuilder{
					SenderPublicKey: harness.config.NodePublicKey(),
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
		harness := newBlockStorageHarness().withSyncNoCommitTimeout(3 * time.Millisecond).start(ctx)

		harness.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).Return(nil, nil).AtLeast(2)

		harness.verifyMocks(t, 2)
	})
}

func TestSyncPetitioner_CompleteSyncFlow(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness().
			withSyncCollectResponsesTimeout(50 * time.Millisecond).
			withSyncCollectChunksTimeout(50 * time.Millisecond).
			withSyncBroadcast(1).
			withValidateConsensusAlgos(4).
			start(ctx)

		// latch until we sent the broadcast (meaning the state machine is now at collecting car state
		require.NoError(t, test.EventuallyVerify(200*time.Millisecond, harness.gossip), "availability response stage failed")

		senderKeyPair := keys.Ed25519KeyPairForTests(7)
		blockAvailabilityResponse := builders.BlockAvailabilityResponseInput().
			WithLastCommittedBlockHeight(primitives.BlockHeight(4)).
			WithFirstBlockHeight(primitives.BlockHeight(1)).
			WithLastBlockHeight(primitives.BlockHeight(4)).
			WithSenderPublicKey(senderKeyPair.PublicKey()).Build()

		// the source key here is the same for both to make our lives easier in BlockSyncResponse
		anotherBlockAvailabilityResponse := builders.BlockAvailabilityResponseInput().
			WithLastCommittedBlockHeight(primitives.BlockHeight(4)).
			WithFirstBlockHeight(primitives.BlockHeight(1)).
			WithLastBlockHeight(primitives.BlockHeight(4)).
			WithSenderPublicKey(senderKeyPair.PublicKey()).Build()

		// fake the collecting car response
		harness.blockStorage.HandleBlockAvailabilityResponse(ctx, blockAvailabilityResponse)
		harness.blockStorage.HandleBlockAvailabilityResponse(ctx, anotherBlockAvailabilityResponse)

		harness.gossip.When("SendBlockSyncRequest", mock.Any, mock.Any).Return(nil, nil).Times(1)

		// latch until we pick a source and request blocks from it
		require.NoError(t, test.EventuallyVerify(200*time.Millisecond, harness.gossip), "availability response stage failed")

		// senderKeyPair must be the same as the chosen BlockAvailabilityResponse
		blockSyncResponse := builders.BlockSyncResponseInput().
			WithSenderPublicKey(senderKeyPair.PublicKey()).
			WithFirstBlockHeight(primitives.BlockHeight(1)).
			WithLastBlockHeight(primitives.BlockHeight(4)).
			WithLastCommittedBlockHeight(primitives.BlockHeight(4)).
			WithSenderPublicKey(senderKeyPair.PublicKey()).Build()

		// fake the response
		harness.blockStorage.HandleBlockSyncResponse(ctx, blockSyncResponse)

		// verify that we committed the blocks
		harness.verifyMocks(t, 4)
	})
}

func TestSyncPetitioner_NeverStartsWhenBlocksAreCommitted(t *testing.T) {
	t.Skip("this test needs to move to CommitBlock unit test, as a 'CommitBlockUpdatesBlockSync'")
	// this test may still be flaky, it runs commits in a busy wait loop that should take longer than the timeout,
	// to make sure we stay at the same state logically.
	// system timing may cause it to flake, but at a very low probability now
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness().
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

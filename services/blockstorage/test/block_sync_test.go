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
		harness := newBlockStorageHarness(t).withSyncNoCommitTimeout(3 * time.Millisecond).start(ctx)

		harness.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).Return(nil, nil).AtLeast(2)

		harness.verifyMocks(t, 2)
	})
}

func TestSyncPetitioner_CompleteSyncFlow(t *testing.T) {
	test.WithContextWithTimeout(1*time.Second, func(ctx context.Context) {

		harness := newBlockStorageHarness(t).
			withSyncNoCommitTimeout(time.Millisecond). // start sync immediately
			withSyncCollectResponsesTimeout(15 * time.Millisecond)

		harness.consensus.Reset().
			When("HandleBlockConsensus", mock.Any, mock.Any).
			Return(&handlers.HandleBlockConsensusOutput{}, nil).
			AtLeast(6) // init, CAR, 4 x block validations

		broadcastBlockAvailabilityRequestLatch := make(chan struct{})
		harness.gossip.
			When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).
			Call(latchingMockHandler(broadcastBlockAvailabilityRequestLatch)).
			AtLeast(1)

		sendBlockSyncRequestLatch := make(chan struct{})
		harness.gossip.
			When("SendBlockSyncRequest", mock.Any, mock.Any).
			Call(latchingMockHandler(sendBlockSyncRequestLatch)).
			AtLeast(1)

		harness.start(ctx)

		requireLatchReleasef(t, ctx, broadcastBlockAvailabilityRequestLatch, "expected sync to collect availability response")

		// fake CAR responses
		syncSourceAddress := keys.EcdsaSecp256K1KeyPairForTests(7)
		blockAvailabilityResponse := buildBlockAvailabilityResponse(syncSourceAddress)
		anotherBlockAvailabilityResponse := buildBlockAvailabilityResponse(syncSourceAddress)

		_, _ = harness.blockStorage.HandleBlockAvailabilityResponse(ctx, blockAvailabilityResponse)
		_, _ = harness.blockStorage.HandleBlockAvailabilityResponse(ctx, anotherBlockAvailabilityResponse)

		requireLatchReleasef(t, ctx, sendBlockSyncRequestLatch, "expected sync to wait for chunks")

		blockSyncResponse := buildBlockSyncResponseInput(syncSourceAddress)
		_, _ = harness.blockStorage.HandleBlockSyncResponse(ctx, blockSyncResponse) // fake blocks

		harness.verifyMocks(t, 1) // verify blocks validated (HandleBlockConsensus x 6)
	})
}

func requireLatchReleasef(t *testing.T, ctx context.Context, latch chan struct{}, format string, args ...interface{}) {
	select {
	case <-latch: // wait on latch
	case <-ctx.Done():
		t.Fatalf(format+"(%v)", append(args, ctx.Err())...)
	}
}

// a helper function which returns a mock call handler which will return nil after notifying calling test of the invocation
func latchingMockHandler(doneChan chan struct{}) func(ctx context.Context, _ interface{}) (*gossiptopics.EmptyOutput, error) {
	return func(ctx context.Context, _ interface{}) (*gossiptopics.EmptyOutput, error) {
		select {
		case doneChan <- struct{}{}:
		case <-ctx.Done():
		}
		return nil, nil
	}
}
func buildBlockSyncResponseInput(senderKeyPair *keys.TestEcdsaSecp256K1KeyPair) *gossiptopics.BlockSyncResponseInput {
	return builders.BlockSyncResponseInput().
		WithSenderNodeAddress(senderKeyPair.NodeAddress()).
		WithFirstBlockHeight(primitives.BlockHeight(1)).
		WithLastBlockHeight(primitives.BlockHeight(4)).
		WithLastCommittedBlockHeight(primitives.BlockHeight(4)).
		WithSenderNodeAddress(senderKeyPair.NodeAddress()).Build()
}

func buildBlockAvailabilityResponse(senderKeyPair *keys.TestEcdsaSecp256K1KeyPair) *gossiptopics.BlockAvailabilityResponseInput {
	return builders.BlockAvailabilityResponseInput().
		WithLastCommittedBlockHeight(primitives.BlockHeight(4)).
		WithFirstBlockHeight(primitives.BlockHeight(1)).
		WithLastBlockHeight(primitives.BlockHeight(4)).
		WithSenderNodeAddress(senderKeyPair.NodeAddress()).Build()
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

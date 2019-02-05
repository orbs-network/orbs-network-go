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
	test.WithContextWithTimeout(1*time.Second, func(ctx context.Context) {

		harness := newBlockStorageHarness(t).
			withSyncNoCommitTimeout(time.Millisecond). // start sync immediately
			withSyncCollectResponsesTimeout(15 * time.Millisecond)

		handleBlockConsensusLatch := latchMockFunction(harness.consensus.Reset(), "HandleBlockConsensus")
		broadcastBlockAvailabilityRequestLatch := latchMockFunction(&harness.gossip.Mock, "BroadcastBlockAvailabilityRequest")
		sendBlockSyncRequestLatch := latchMockFunction(&harness.gossip.Mock, "SendBlockSyncRequest")

		go harness.start(ctx) // go because start() will block until next line is reached
		requireMockFunctionLatchTriggerf(t, ctx, handleBlockConsensusLatch, "expected service to notify sync with consensus algo on init")

		requireMockFunctionLatchTriggerf(t, ctx, handleBlockConsensusLatch, "expected sync to notify consensus algo of current height")
		requireMockFunctionLatchTriggerf(t, ctx, broadcastBlockAvailabilityRequestLatch, "expected sync to collect availability response")

		// fake CAR responses
		syncSourceAddress := keys.EcdsaSecp256K1KeyPairForTests(7)
		blockAvailabilityResponse := buildBlockAvailabilityResponse(syncSourceAddress)
		anotherBlockAvailabilityResponse := buildBlockAvailabilityResponse(syncSourceAddress)

		go harness.blockStorage.HandleBlockAvailabilityResponse(ctx, blockAvailabilityResponse)
		go harness.blockStorage.HandleBlockAvailabilityResponse(ctx, anotherBlockAvailabilityResponse)

		requireMockFunctionLatchTriggerf(t, ctx, sendBlockSyncRequestLatch, "expected sync to wait for chunks")

		numOfBlocks := 4
		blockSyncResponse := buildBlockSyncResponseInput(syncSourceAddress, numOfBlocks)
		go harness.blockStorage.HandleBlockSyncResponse(ctx, blockSyncResponse) // fake block sync response

		for i := 1; i <= numOfBlocks; i++ {
			requireMockFunctionLatchTriggerf(t, ctx, handleBlockConsensusLatch, "expected block %d to be validated on commit", i)
		}
	})
}

// a helper function which returns a channel used for syncing test code on mock function calls.
// this implementation is compatible only with mock functions receiving a context and one additional argument,
// and returning two arguments. after calling this method, each invocation of this mock function will block until
// the test code reads from the latch channel, or the context terminates.
func latchMockFunction(m *mock.Mock, name string) <-chan struct{} {
	latch := make(chan struct{})
	m.When(name, mock.Any, mock.Any).
		Call(func(ctx context.Context, _ interface{}) (interface{}, interface{}) {
			select {
			case latch <- struct{}{}:
			case <-ctx.Done():
			}
			return nil, nil
		})
	return latch
}

// a helper function which works with latch channels returned from latchMockFunction.
// test code should use this helper to sync with mock function invocations.
// this function blocks until a single invocation of the mock function tied to the latch channel occurs.
func requireMockFunctionLatchTriggerf(t *testing.T, ctx context.Context, latch <-chan struct{}, format string, args ...interface{}) {
	select {
	case <-latch: // wait on latch
	case <-ctx.Done():
		t.Fatalf(format+"(%v)", append(args, ctx.Err())...)
	}
}

func buildBlockSyncResponseInput(senderKeyPair *keys.TestEcdsaSecp256K1KeyPair, numOfBlocks int) *gossiptopics.BlockSyncResponseInput {
	return builders.BlockSyncResponseInput().
		WithSenderNodeAddress(senderKeyPair.NodeAddress()).
		WithFirstBlockHeight(primitives.BlockHeight(1)).
		WithLastBlockHeight(primitives.BlockHeight(numOfBlocks)).
		WithLastCommittedBlockHeight(primitives.BlockHeight(numOfBlocks)).
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

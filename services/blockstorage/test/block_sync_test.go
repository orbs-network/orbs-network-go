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

func TestSyncSourceHandlesBlockAvailabilityRequest(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newHarness(ctx)

		harness.expectCommitStateDiffTimes(2)

		harness.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(1)).WithBlockCreated(time.Now()).Build())
		harness.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(2)).WithBlockCreated(time.Now()).Build())

		senderKeyPair := keys.Ed25519KeyPairForTests(9)

		input := builders.BlockAvailabilityRequestInput().WithLastCommittedBlockHeight(primitives.BlockHeight(0)).WithSenderPublicKey(senderKeyPair.PublicKey()).Build()

		response := builders.BlockAvailabilityResponseInput().
			WithLastCommittedBlockHeight(primitives.BlockHeight(2)).
			WithFirstBlockHeight(primitives.BlockHeight(1)).
			WithLastBlockHeight(primitives.BlockHeight(2)).
			WithSenderPublicKey(harness.config.NodePublicKey()).
			WithRecipientPublicKey(senderKeyPair.PublicKey()).Build()

		harness.gossip.When("SendBlockAvailabilityResponse", response).Return(nil, nil).Times(1)

		_, err := harness.blockStorage.HandleBlockAvailabilityRequest(input)
		require.NoError(t, err)

		harness.verifyMocks(t, 2)
	})
}

func TestSyncSourceHandlesBlockSyncRequest(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newHarness(ctx)

		harness.expectCommitStateDiffTimes(4)

		blocks := []*protocol.BlockPairContainer{
			builders.BlockPair().WithHeight(primitives.BlockHeight(1)).WithBlockCreated(time.Now()).Build(),
			builders.BlockPair().WithHeight(primitives.BlockHeight(2)).WithBlockCreated(time.Now()).Build(),
			builders.BlockPair().WithHeight(primitives.BlockHeight(3)).WithBlockCreated(time.Now()).Build(),
			builders.BlockPair().WithHeight(primitives.BlockHeight(4)).WithBlockCreated(time.Now()).Build(),
		}

		harness.commitBlock(blocks[0])
		harness.commitBlock(blocks[1])
		harness.commitBlock(blocks[2])
		harness.commitBlock(blocks[3])

		expectedBlocks := []*protocol.BlockPairContainer{blocks[1], blocks[2], blocks[3]}

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
					LastBlockHeight:          primitives.BlockHeight(4),
					LastCommittedBlockHeight: primitives.BlockHeight(4),
				}).Build(),
				BlockPairs: expectedBlocks,
			},
		}

		harness.gossip.When("SendBlockSyncResponse", response).Return(nil, nil).Times(1)

		_, err := harness.blockStorage.HandleBlockSyncRequest(input)
		require.NoError(t, err)

		harness.verifyMocks(t, 4)
	})
}

// TODO move to unit tests
func TestSyncSourceIgnoresRangesOfBlockSyncRequestAccordingToLocalBatchSettings(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newHarness(ctx)
		harness.setBatchSize(2)

		harness.expectCommitStateDiffTimes(4)

		blocks := []*protocol.BlockPairContainer{
			builders.BlockPair().WithHeight(primitives.BlockHeight(1)).WithBlockCreated(time.Now()).Build(),
			builders.BlockPair().WithHeight(primitives.BlockHeight(2)).WithBlockCreated(time.Now()).Build(),
			builders.BlockPair().WithHeight(primitives.BlockHeight(3)).WithBlockCreated(time.Now()).Build(),
			builders.BlockPair().WithHeight(primitives.BlockHeight(4)).WithBlockCreated(time.Now()).Build(),
		}

		harness.commitBlock(blocks[0])
		harness.commitBlock(blocks[1])
		harness.commitBlock(blocks[2])
		harness.commitBlock(blocks[3])

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

		harness.gossip.When("SendBlockSyncResponse", response).Return(nil, nil).Times(1)

		_, err := harness.blockStorage.HandleBlockSyncRequest(input)
		require.NoError(t, err)

		harness.verifyMocks(t, 4)
	})
}

func TestSyncPetitionerBroadcastsBlockAvailabilityRequest(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newHarness(ctx)

		harness.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any).Return(nil, nil).AtLeast(2)

		harness.verifyMocks(t, 2)
	})
}

func TestSyncCompletePetitionerSyncFlow(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newHarness(ctx)

		harness.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any).Return(nil, nil).AtLeast(1)

		senderKeyPair := keys.Ed25519KeyPairForTests(7)

		blockAvailabilityResponse := builders.BlockAvailabilityResponseInput().
			WithLastCommittedBlockHeight(primitives.BlockHeight(4)).
			WithFirstBlockHeight(primitives.BlockHeight(1)).
			WithLastBlockHeight(primitives.BlockHeight(4)).Build()

		anotherSenderKeyPair := keys.Ed25519KeyPairForTests(8)
		anotherBlockAvailabilityResponse := builders.BlockAvailabilityResponseInput().
			WithLastCommittedBlockHeight(primitives.BlockHeight(3)).
			WithFirstBlockHeight(primitives.BlockHeight(1)).
			WithLastBlockHeight(primitives.BlockHeight(3)).
			WithSenderPublicKey(anotherSenderKeyPair.PublicKey()).Build()

		harness.gossip.When("SendBlockSyncRequest", mock.Any).Return(nil, nil).Times(1)
		harness.blockStorage.HandleBlockAvailabilityResponse(blockAvailabilityResponse)
		harness.blockStorage.HandleBlockAvailabilityResponse(anotherBlockAvailabilityResponse)

		// we use this to verify that the SendBlockSyncRequest was sent out, meaning that the AvailabilityResponse stage is done
		// we saw it can take up to 15ms in some cases (although it is suppose to take just 1ms this is the docker timing delay we see in CI)
		test.EventuallyVerify(50*time.Millisecond, harness.gossip)

		blockSyncResponse := builders.BlockSyncResponseInput().
			WithFirstBlockHeight(primitives.BlockHeight(1)).
			WithLastBlockHeight(primitives.BlockHeight(4)).
			WithLastCommittedBlockHeight(primitives.BlockHeight(4)).
			WithSenderPublicKey(senderKeyPair.PublicKey()).Build()

		harness.expectCommitStateDiffTimes(4)
		harness.expectValidateWithConsensusAlgosTimes(4)

		harness.blockStorage.HandleBlockSyncResponse(blockSyncResponse)

		time.Sleep(1 * time.Millisecond)

		harness.verifyMocks(t, 4)
	})
}

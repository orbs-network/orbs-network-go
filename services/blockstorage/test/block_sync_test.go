package test

import (
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func generateBlockAvailabilityRequestInput(lastCommittedBlockHeight primitives.BlockHeight, senderPublicKey primitives.Ed25519PublicKey) *gossiptopics.BlockAvailabilityRequestInput {
	return &gossiptopics.BlockAvailabilityRequestInput{
		Message: &gossipmessages.BlockAvailabilityRequestMessage{
			SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
				LastCommittedBlockHeight: lastCommittedBlockHeight,
			}).Build(),
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: senderPublicKey,
			}).Build(),
		},
	}
}

func TestSyncSourceHandlesBlockAvailabilityRequest(t *testing.T) {
	// FIXME user WithContext everywhere
	//test.WithContext(func(ctx context.Context) {

	harness := newHarness()

	harness.expectCommitStateDiffTimes(2)

	harness.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(1)).WithBlockCreated(time.Now()).Build())
	harness.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(2)).WithBlockCreated(time.Now()).Build())

	senderKeyPair := keys.Ed25519KeyPairForTests(9)

	input := generateBlockAvailabilityRequestInput(primitives.BlockHeight(0), senderKeyPair.PublicKey())
	response := builders.BlockAvailabilityResponseInput().
		WithLastCommittedBlockHeight(primitives.BlockHeight(2)).
		WithFirstBlockHeight(primitives.BlockHeight(1)).
		WithLastBlockHeight(primitives.BlockHeight(2)).
		WithSenderPublicKey(harness.config.NodePublicKey()).
		WithRecipientPublicKey(senderKeyPair.PublicKey()).Build()

	harness.gossip.When("SendBlockAvailabilityResponse", response).Return(nil, nil).Times(1)

	_, err := harness.blockStorage.HandleBlockAvailabilityRequest(input)
	require.NoError(t, err)

	harness.verifyMocks(t)
}

func TestSyncSourceIgnoresBlockAvailabilityRequestIfNoBlocksWereCommitted(t *testing.T) {
	harness := newHarness()

	senderKeyPair := keys.Ed25519KeyPairForTests(9)
	input := generateBlockAvailabilityRequestInput(primitives.BlockHeight(2), senderKeyPair.PublicKey())

	harness.gossip.When("SendBlockAvailabilityResponse", mock.Any).Return(nil, nil).Times(0)

	_, err := harness.blockStorage.HandleBlockAvailabilityRequest(input)
	require.NoError(t, err)

	harness.verifyMocks(t)
}

func TestSyncSourceIgnoresBlockAvailabilityRequestIfPetitionerIsFurtherAhead(t *testing.T) {
	harness := newHarness()

	harness.expectCommitStateDiffTimes(2)

	harness.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(1)).WithBlockCreated(time.Now()).Build())
	harness.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(2)).WithBlockCreated(time.Now()).Build())

	senderKeyPair := keys.Ed25519KeyPairForTests(9)
	input := generateBlockAvailabilityRequestInput(primitives.BlockHeight(1972), senderKeyPair.PublicKey())

	harness.gossip.When("SendBlockAvailabilityResponse", mock.Any).Return(nil, nil).Times(0)

	_, err := harness.blockStorage.HandleBlockAvailabilityRequest(input)
	require.NoError(t, err)

	harness.verifyMocks(t)
}

func TestSyncPetitionerHandlesBlockAvailabilityResponse(t *testing.T) {
	harness := newHarness()

	harness.expectCommitStateDiffTimes(2)

	harness.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(1)).WithBlockCreated(time.Now()).Build())
	harness.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(2)).WithBlockCreated(time.Now()).Build())

	input := builders.BlockAvailabilityResponseInput().
		WithLastCommittedBlockHeight(primitives.BlockHeight(999)).
		WithFirstBlockHeight(primitives.BlockHeight(0)).
		WithLastBlockHeight(primitives.BlockHeight(0)).Build()

	request := &gossiptopics.BlockSyncRequestInput{
		RecipientPublicKey: input.Message.Sender.SenderPublicKey(),
		Message: &gossipmessages.BlockSyncRequestMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: harness.config.NodePublicKey(),
			}).Build(),
			SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
				LastBlockHeight:          primitives.BlockHeight(10002),
				FirstBlockHeight:         primitives.BlockHeight(3),
				LastCommittedBlockHeight: primitives.BlockHeight(2),
			}).Build(),
		},
	}

	harness.gossip.When("SendBlockSyncRequest", request).Return(nil, nil).Times(1)

	_, err := harness.blockStorage.HandleBlockAvailabilityResponse(input)
	require.NoError(t, err)

	harness.verifyMocks(t)
}

func TestSyncPetitionerIgnoresBlockAvailabilityResponseIfAlreadyInSync(t *testing.T) {
	harness := newHarness()

	harness.expectCommitStateDiffTimes(2)

	harness.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(1)).WithBlockCreated(time.Now()).Build())
	harness.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(2)).WithBlockCreated(time.Now()).Build())

	input := builders.BlockAvailabilityResponseInput().
		WithLastCommittedBlockHeight(primitives.BlockHeight(2)).
		WithFirstBlockHeight(primitives.BlockHeight(1)).
		WithLastBlockHeight(primitives.BlockHeight(2)).Build()

	harness.gossip.When("SendBlockSyncRequest", mock.Any).Return(nil, nil).Times(0)

	_, err := harness.blockStorage.HandleBlockAvailabilityResponse(input)
	require.NoError(t, err)

	harness.verifyMocks(t)
}

func TestSyncPetitionerHandlesBlockAvailabilityResponseFromMultipleSources(t *testing.T) {
	harness := newHarness()

	harness.expectCommitStateDiffTimes(2)

	harness.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(1)).WithBlockCreated(time.Now()).Build())
	harness.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(2)).WithBlockCreated(time.Now()).Build())

	senderKeyPair := keys.Ed25519KeyPairForTests(9)
	input := builders.BlockAvailabilityResponseInput().
		WithLastCommittedBlockHeight(primitives.BlockHeight(2)).
		WithFirstBlockHeight(primitives.BlockHeight(1)).
		WithLastBlockHeight(primitives.BlockHeight(2)).
		WithSenderPublicKey(senderKeyPair.PublicKey()).Build()

	anotherSenderKeyPair := keys.Ed25519KeyPairForTests(8)
	anotherInput := builders.BlockAvailabilityResponseInput().
		WithLastCommittedBlockHeight(primitives.BlockHeight(3)).
		WithFirstBlockHeight(primitives.BlockHeight(1)).
		WithLastBlockHeight(primitives.BlockHeight(3)).
		WithSenderPublicKey(anotherSenderKeyPair.PublicKey()).Build()

	harness.gossip.When("SendBlockSyncRequest", mock.Any).Return(nil, nil).Times(1)

	_, err := harness.blockStorage.HandleBlockAvailabilityResponse(input)
	require.NoError(t, err)

	_, err = harness.blockStorage.HandleBlockAvailabilityResponse(anotherInput)
	require.NoError(t, err)

	harness.verifyMocks(t)
}

func generateBlockSyncRequestInput(lastBlockHeight primitives.BlockHeight, desirableBlockHeight primitives.BlockHeight, senderPublicKey primitives.Ed25519PublicKey) *gossiptopics.BlockSyncRequestInput {
	return &gossiptopics.BlockSyncRequestInput{
		Message: &gossipmessages.BlockSyncRequestMessage{
			SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
				FirstBlockHeight:         lastBlockHeight,
				LastBlockHeight:          desirableBlockHeight,
				LastCommittedBlockHeight: lastBlockHeight,
			}).Build(),
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: senderPublicKey,
			}).Build(),
		},
	}
}

func TestSyncSourceHandlesBlockSyncRequest(t *testing.T) {
	harness := newHarness()

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
	input := generateBlockSyncRequestInput(primitives.BlockHeight(2), primitives.BlockHeight(10002), senderKeyPair.PublicKey())

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

	harness.verifyMocks(t)
}

func TestSyncSourceIgnoresRangesOfBlockSyncRequestAccordingToLocalBatchSettings(t *testing.T) {
	harness := newHarness()
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
	input := generateBlockSyncRequestInput(primitives.BlockHeight(2), primitives.BlockHeight(10002), senderKeyPair.PublicKey())

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

	harness.verifyMocks(t)
}

// FIXME use builders instead
func generateBlockSyncResponseInput(lastBlockHeight primitives.BlockHeight, desirableBlockHeight primitives.BlockHeight, senderPublicKey primitives.Ed25519PublicKey) *gossiptopics.BlockSyncResponseInput {
	var blocks []*protocol.BlockPairContainer

	for i := lastBlockHeight; i <= desirableBlockHeight; i++ {
		blocks = append(blocks, builders.BlockPair().WithHeight(i).WithBlockCreated(time.Now()).Build())
	}

	return &gossiptopics.BlockSyncResponseInput{
		Message: &gossipmessages.BlockSyncResponseMessage{
			SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
				FirstBlockHeight:         lastBlockHeight,
				LastBlockHeight:          desirableBlockHeight,
				LastCommittedBlockHeight: lastBlockHeight,
			}).Build(),
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: senderPublicKey,
			}).Build(),
			BlockPairs: blocks,
		},
	}
}

func TestSyncPetitionerHandlesBlockSyncResponse(t *testing.T) {
	harness := newHarness()

	harness.expectCommitStateDiffTimes(4)
	harness.expectValidateWithConsensusAlgosTimes(2)

	harness.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(1)).WithBlockCreated(time.Now()).Build())
	harness.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(2)).WithBlockCreated(time.Now()).Build())

	senderKeyPair := keys.Ed25519KeyPairForTests(9)
	input := generateBlockSyncResponseInput(primitives.BlockHeight(3), primitives.BlockHeight(4), senderKeyPair.PublicKey())

	_, err := harness.blockStorage.HandleBlockSyncResponse(input)
	require.NoError(t, err)

	harness.verifyMocks(t)
}

func TestSyncPetitionerHandlesBlockSyncResponseFromMultipleSenders(t *testing.T) {
	harness := newHarness()

	harness.expectCommitStateDiffTimes(4)
	harness.expectValidateWithConsensusAlgosTimes(3)

	harness.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(1)).WithBlockCreated(time.Now()).Build())
	harness.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(2)).WithBlockCreated(time.Now()).Build())

	senderKeyPair := keys.Ed25519KeyPairForTests(7)
	input := generateBlockSyncResponseInput(primitives.BlockHeight(3), primitives.BlockHeight(4), senderKeyPair.PublicKey())

	anotherSenderKeyPair := keys.Ed25519KeyPairForTests(8)
	inputFromAnotherSender := generateBlockSyncResponseInput(primitives.BlockHeight(3), primitives.BlockHeight(5), anotherSenderKeyPair.PublicKey())

	_, err := harness.blockStorage.HandleBlockSyncResponse(input)
	require.NoError(t, err)

	_, err = harness.blockStorage.HandleBlockSyncResponse(inputFromAnotherSender)
	require.NoError(t, err)

	lastCommittedBlockheight, _ := harness.blockStorage.GetLastCommittedBlockHeight(&services.GetLastCommittedBlockHeightInput{})
	require.Equal(t, primitives.BlockHeight(4), lastCommittedBlockheight.LastCommittedBlockHeight)

	harness.verifyMocks(t)
}

// FIXME implement
func TestSyncPetitionerHandlesBlockSyncResponseAndRunsValidationChecks(t *testing.T) {
	t.Skip("not implemented")
}

func TestSyncPetitionerBroadcastsBlockAvailabilityRequest(t *testing.T) {
	harness := newHarness()

	harness.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any).Return(nil, nil).AtLeast(2)

	harness.verifyMocks(t)
}

func TestSyncCompletePetitionerSyncFlow(t *testing.T) {
	harness := newHarness()

	harness.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any).Return(nil, nil).AtLeast(1)

	senderKeyPair := keys.Ed25519KeyPairForTests(7)

	blockAvailabilityResponse := builders.BlockAvailabilityResponseInput().
		WithLastCommittedBlockHeight(primitives.BlockHeight(4)).
		WithFirstBlockHeight(primitives.BlockHeight(1)).
		WithLastBlockHeight(primitives.BlockHeight(4)).Build()

	harness.gossip.When("SendBlockSyncRequest", mock.Any).Return(nil, nil).AtLeast(1)
	harness.blockStorage.HandleBlockAvailabilityResponse(blockAvailabilityResponse)

	time.Sleep(4 * time.Millisecond)

	blockSyncResponse := generateBlockSyncResponseInput(primitives.BlockHeight(1), primitives.BlockHeight(4), senderKeyPair.PublicKey())

	harness.expectCommitStateDiffTimes(4)
	harness.expectValidateWithConsensusAlgosTimes(4)

	harness.blockStorage.HandleBlockSyncResponse(blockSyncResponse)

	time.Sleep(1 * time.Millisecond)

	harness.verifyMocks(t)
}

func TestSyncCompleteSourceSyncFlow(t *testing.T) {
	t.Skip("not implemented")
}

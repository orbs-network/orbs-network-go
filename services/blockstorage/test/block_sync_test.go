package test

import (
	"github.com/orbs-network/go-mock"
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

	driver := NewDriver()

	driver.expectCommitStateDiffTimes(2)

	driver.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(1)).WithBlockCreated(time.Now()).Build())
	driver.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(2)).WithBlockCreated(time.Now()).Build())

	senderKeyPair := keys.Ed25519KeyPairForTests(9)

	input := generateBlockAvailabilityRequestInput(primitives.BlockHeight(0), senderKeyPair.PublicKey())
	response := &gossiptopics.BlockAvailabilityResponseInput{
		RecipientPublicKey: senderKeyPair.PublicKey(),
		Message: &gossipmessages.BlockAvailabilityResponseMessage{
			SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
				LastBlockHeight:          primitives.BlockHeight(2),
				FirstBlockHeight:         primitives.BlockHeight(1),
				LastCommittedBlockHeight: primitives.BlockHeight(2),
			}).Build(),
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: driver.config.NodePublicKey(),
			}).Build(),
		},
	}

	driver.gossip.When("SendBlockAvailabilityResponse", response).Return(nil, nil).Times(1)

	_, err := driver.blockStorage.HandleBlockAvailabilityRequest(input)
	require.NoError(t, err)

	// FIXME remove sleep
	time.Sleep(1 * time.Millisecond)

	driver.verifyMocks(t)
}

func TestSyncSourceIgnoresBlockAvailabilityRequestIfNoBlocksWereCommitted(t *testing.T) {
	driver := NewDriver()

	senderKeyPair := keys.Ed25519KeyPairForTests(9)
	input := generateBlockAvailabilityRequestInput(primitives.BlockHeight(2), senderKeyPair.PublicKey())

	driver.gossip.When("SendBlockAvailabilityResponse", mock.Any).Return(nil, nil).Times(0)

	_, err := driver.blockStorage.HandleBlockAvailabilityRequest(input)
	require.NoError(t, err)

	// FIXME remove sleep
	time.Sleep(1 * time.Millisecond)

	driver.verifyMocks(t)
}

func TestSyncSourceIgnoresBlockAvailabilityRequestIfPetitionerIsFurtherAhead(t *testing.T) {
	driver := NewDriver()

	driver.expectCommitStateDiffTimes(2)

	driver.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(1)).WithBlockCreated(time.Now()).Build())
	driver.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(2)).WithBlockCreated(time.Now()).Build())

	senderKeyPair := keys.Ed25519KeyPairForTests(9)
	input := generateBlockAvailabilityRequestInput(primitives.BlockHeight(1972), senderKeyPair.PublicKey())

	driver.gossip.When("SendBlockAvailabilityResponse", mock.Any).Return(nil, nil).Times(0)

	_, err := driver.blockStorage.HandleBlockAvailabilityRequest(input)
	require.NoError(t, err)

	// FIXME remove sleep
	time.Sleep(1 * time.Millisecond)

	driver.verifyMocks(t)
}

func generateBlockAvailabilityResponseInput(lastCommittedBlockHeight primitives.BlockHeight, senderPublicKey primitives.Ed25519PublicKey) *gossiptopics.BlockAvailabilityResponseInput {
	return &gossiptopics.BlockAvailabilityResponseInput{
		Message: &gossipmessages.BlockAvailabilityResponseMessage{
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

func TestSyncPetitionerHandlesBlockAvailabilityResponse(t *testing.T) {
	driver := NewDriver()

	driver.expectCommitStateDiffTimes(2)
	driver.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(1)).WithBlockCreated(time.Now()).Build())
	driver.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(2)).WithBlockCreated(time.Now()).Build())

	senderKeyPair := keys.Ed25519KeyPairForTests(9)
	input := generateBlockAvailabilityResponseInput(primitives.BlockHeight(999), senderKeyPair.PublicKey())

	request := &gossiptopics.BlockSyncRequestInput{
		RecipientPublicKey: input.Message.Sender.SenderPublicKey(),
		Message: &gossipmessages.BlockSyncRequestMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: driver.config.NodePublicKey(),
			}).Build(),
			SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
				LastBlockHeight:          primitives.BlockHeight(10002),
				FirstBlockHeight:         primitives.BlockHeight(3),
				LastCommittedBlockHeight: primitives.BlockHeight(2),
			}).Build(),
		},
	}

	driver.gossip.When("SendBlockSyncRequest", request).Return(nil, nil).Times(1)

	_, err := driver.blockStorage.HandleBlockAvailabilityResponse(input)
	require.NoError(t, err)

	// FIXME remove sleep
	time.Sleep(1 * time.Millisecond)

	driver.verifyMocks(t)
}

func TestSyncPetitionerIgnoresBlockAvailabilityResponseIfAlreadyInSync(t *testing.T) {
	driver := NewDriver()

	driver.expectCommitStateDiffTimes(2)
	driver.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(1)).WithBlockCreated(time.Now()).Build())
	driver.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(2)).WithBlockCreated(time.Now()).Build())

	senderKeyPair := keys.Ed25519KeyPairForTests(9)
	input := generateBlockAvailabilityResponseInput(primitives.BlockHeight(2), senderKeyPair.PublicKey())

	driver.gossip.When("SendBlockSyncRequest", mock.Any).Return(nil, nil).Times(0)

	_, err := driver.blockStorage.HandleBlockAvailabilityResponse(input)
	require.NoError(t, err)

	// FIXME remove sleep
	time.Sleep(1 * time.Millisecond)

	driver.verifyMocks(t)
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
	driver := NewDriver()

	driver.expectCommitStateDiffTimes(4)

	blocks := []*protocol.BlockPairContainer{
		builders.BlockPair().WithHeight(primitives.BlockHeight(1)).WithBlockCreated(time.Now()).Build(),
		builders.BlockPair().WithHeight(primitives.BlockHeight(2)).WithBlockCreated(time.Now()).Build(),
		builders.BlockPair().WithHeight(primitives.BlockHeight(3)).WithBlockCreated(time.Now()).Build(),
		builders.BlockPair().WithHeight(primitives.BlockHeight(4)).WithBlockCreated(time.Now()).Build(),
	}

	driver.commitBlock(blocks[0])
	driver.commitBlock(blocks[1])
	driver.commitBlock(blocks[2])
	driver.commitBlock(blocks[3])

	expectedBlocks := []*protocol.BlockPairContainer{blocks[1], blocks[2], blocks[3]}

	senderKeyPair := keys.Ed25519KeyPairForTests(9)
	input := generateBlockSyncRequestInput(primitives.BlockHeight(2), primitives.BlockHeight(10002), senderKeyPair.PublicKey())

	response := &gossiptopics.BlockSyncResponseInput{
		RecipientPublicKey: senderKeyPair.PublicKey(),
		Message: &gossipmessages.BlockSyncResponseMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: driver.config.NodePublicKey(),
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

	driver.gossip.When("SendBlockSyncResponse", response).Return(nil, nil).Times(1)

	_, err := driver.blockStorage.HandleBlockSyncRequest(input)
	require.NoError(t, err)

	// FIXME remove sleep
	time.Sleep(1 * time.Millisecond)

	driver.verifyMocks(t)
}

func TestSyncSourceIgnoresRangesOfBlockSyncRequestAccordingToLocalBatchSettings(t *testing.T) {
	driver := NewDriver()
	driver.setBatchSize(2)

	driver.expectCommitStateDiffTimes(4)

	blocks := []*protocol.BlockPairContainer{
		builders.BlockPair().WithHeight(primitives.BlockHeight(1)).WithBlockCreated(time.Now()).Build(),
		builders.BlockPair().WithHeight(primitives.BlockHeight(2)).WithBlockCreated(time.Now()).Build(),
		builders.BlockPair().WithHeight(primitives.BlockHeight(3)).WithBlockCreated(time.Now()).Build(),
		builders.BlockPair().WithHeight(primitives.BlockHeight(4)).WithBlockCreated(time.Now()).Build(),
	}

	driver.commitBlock(blocks[0])
	driver.commitBlock(blocks[1])
	driver.commitBlock(blocks[2])
	driver.commitBlock(blocks[3])

	expectedBlocks := []*protocol.BlockPairContainer{blocks[1], blocks[2]}

	senderKeyPair := keys.Ed25519KeyPairForTests(9)
	input := generateBlockSyncRequestInput(primitives.BlockHeight(2), primitives.BlockHeight(10002), senderKeyPair.PublicKey())

	response := &gossiptopics.BlockSyncResponseInput{
		RecipientPublicKey: senderKeyPair.PublicKey(),
		Message: &gossipmessages.BlockSyncResponseMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: driver.config.NodePublicKey(),
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

	driver.gossip.When("SendBlockSyncResponse", response).Return(nil, nil).Times(1)

	_, err := driver.blockStorage.HandleBlockSyncRequest(input)
	require.NoError(t, err)

	// FIXME remove sleep
	time.Sleep(1 * time.Millisecond)

	driver.verifyMocks(t)
}

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
	driver := NewDriver()

	driver.expectCommitStateDiffTimes(4)

	driver.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(1)).WithBlockCreated(time.Now()).Build())
	driver.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(2)).WithBlockCreated(time.Now()).Build())

	senderKeyPair := keys.Ed25519KeyPairForTests(9)
	input := generateBlockSyncResponseInput(primitives.BlockHeight(3), primitives.BlockHeight(4), senderKeyPair.PublicKey())

	_, err := driver.blockStorage.HandleBlockSyncResponse(input)
	require.NoError(t, err)

	// FIXME remove sleep
	time.Sleep(1 * time.Millisecond)

	driver.verifyMocks(t)
}

func TestSyncPetitionerHandlesBlockSyncResponseFromMultipleSenders(t *testing.T) {
	driver := NewDriver()

	driver.expectCommitStateDiffTimes(5)

	driver.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(1)).WithBlockCreated(time.Now()).Build())
	driver.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(2)).WithBlockCreated(time.Now()).Build())

	senderKeyPair := keys.Ed25519KeyPairForTests(7)
	input := generateBlockSyncResponseInput(primitives.BlockHeight(3), primitives.BlockHeight(4), senderKeyPair.PublicKey())

	anotherSenderKeyPair := keys.Ed25519KeyPairForTests(8)
	inputFromAnotherSender := generateBlockSyncResponseInput(primitives.BlockHeight(3), primitives.BlockHeight(5), anotherSenderKeyPair.PublicKey())

	_, err := driver.blockStorage.HandleBlockSyncResponse(input)
	require.NoError(t, err)

	_, err = driver.blockStorage.HandleBlockSyncResponse(inputFromAnotherSender)
	require.NoError(t, err)

	// FIXME remove sleep
	time.Sleep(1 * time.Millisecond)

	driver.verifyMocks(t)
}

// FIXME implement
func TestSyncPetitionerHandlesBlockSyncResponseAndRunsValidationChecks(t *testing.T) {
	t.Skip("not implemented")
}

func TestSyncPetitionerBroadcastsBlockAvailabilityRequest(t *testing.T) {
	driver := NewDriver()

	driver.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any).Return(nil, nil).AtLeast(2)

	time.Sleep(6 * time.Millisecond)

	driver.verifyMocks(t)
}

func TestSyncCompletePetitionerSyncFlow(t *testing.T) {
	driver := NewDriver()

	driver.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any).Return(nil, nil).AtLeast(2)

	time.Sleep(4 * time.Millisecond)

	senderKeyPair := keys.Ed25519KeyPairForTests(7)

	blockAvailabilityResponse := generateBlockAvailabilityResponseInput(primitives.BlockHeight(4), senderKeyPair.PublicKey())

	driver.gossip.When("SendBlockSyncRequest", mock.Any).Return(nil, nil).AtLeast(1)
	driver.blockStorage.HandleBlockAvailabilityResponse(blockAvailabilityResponse)

	blockSyncResponse := generateBlockSyncResponseInput(primitives.BlockHeight(1), primitives.BlockHeight(4), senderKeyPair.PublicKey())

	driver.expectCommitStateDiffTimes(4)
	driver.blockStorage.HandleBlockSyncResponse(blockSyncResponse)

	time.Sleep(1 * time.Millisecond)

	driver.verifyMocks(t)
}

func TestSyncCompleteSourceSyncFlow(t *testing.T) {
	t.Skip("not implemented")
}

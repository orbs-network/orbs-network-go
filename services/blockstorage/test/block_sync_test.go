package test

import (
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"testing"
	"time"
)

func generateBlockAvailabilityRequestInput(lastCommittedBlockHeight primitives.BlockHeight, senderPublicKey primitives.Ed25519PublicKey) *gossiptopics.BlockAvailabilityRequestInput {
	return &gossiptopics.BlockAvailabilityRequestInput{
		Message: &gossipmessages.BlockAvailabilityRequestMessage{
			SignedRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
				LastCommittedBlockHeight: lastCommittedBlockHeight,
			}).Build(),
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: senderPublicKey,
			}).Build(),
		},
	}
}

func TestSyncHandleBlockAvailabilityRequest(t *testing.T) {
	driver := NewDriver()

	driver.expectCommitStateDiffTimes(2)

	driver.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(1)).WithBlockCreated(time.Now()).Build())
	driver.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(2)).WithBlockCreated(time.Now()).Build())

	senderKeyPair := keys.Ed25519KeyPairForTests(9)

	input := generateBlockAvailabilityRequestInput(primitives.BlockHeight(0), senderKeyPair.PublicKey())
	response := &gossiptopics.BlockAvailabilityResponseInput{
		RecipientPublicKey: senderKeyPair.PublicKey(),
		Message: &gossipmessages.BlockAvailabilityResponseMessage{
			SignedRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                 gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
				LastAvailableBlockHeight:  primitives.BlockHeight(2),
				FirstAvailableBlockHeight: primitives.BlockHeight(1),
				LastCommittedBlockHeight:  primitives.BlockHeight(2),
			}).Build(),
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: driver.config.NodePublicKey(),
			}).Build(),
		},
	}

	driver.blockSync.When("SendBlockAvailabilityResponse", response).Return(nil, nil).Times(1)

	driver.blockStorage.HandleBlockAvailabilityRequest(input)

	driver.verifyMocks(t)
}

func TestSyncHandleBlockAvailabilityRequestIgnoredIfNoBlocksWereCommitted(t *testing.T) {
	driver := NewDriver()

	senderKeyPair := keys.Ed25519KeyPairForTests(9)
	input := generateBlockAvailabilityRequestInput(primitives.BlockHeight(2), senderKeyPair.PublicKey())

	driver.blockSync.When("SendBlockAvailabilityResponse", mock.Any).Return(nil, nil).Times(0)

	driver.blockStorage.HandleBlockAvailabilityRequest(input)

	driver.verifyMocks(t)
}

func TestSyncHandleBlockAvailabilityRequestIgnoredIfSenderIsInSync(t *testing.T) {
	driver := NewDriver()

	driver.expectCommitStateDiffTimes(2)

	driver.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(1)).WithBlockCreated(time.Now()).Build())
	driver.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(2)).WithBlockCreated(time.Now()).Build())

	senderKeyPair := keys.Ed25519KeyPairForTests(9)
	input := generateBlockAvailabilityRequestInput(primitives.BlockHeight(1972), senderKeyPair.PublicKey())

	driver.blockSync.When("SendBlockAvailabilityResponse", mock.Any).Return(nil, nil).Times(0)

	driver.blockStorage.HandleBlockAvailabilityRequest(input)

	driver.verifyMocks(t)
}

package test

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"testing"
	"time"
)

func TestSyncHandleBlockAvailabilityRequest(t *testing.T) {
	driver := NewDriver()

	senderKeyPair := keys.Ed25519KeyPairForTests(9)

	input := &gossiptopics.BlockAvailabilityRequestInput{
		Message: &gossipmessages.BlockAvailabilityRequestMessage{
			SignedRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
				LastCommittedBlockHeight: primitives.BlockHeight(0),
			}).Build(),
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: senderKeyPair.PublicKey(),
			}).Build(),
		},
	}

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

	driver.expectCommitStateDiffTimes(2)

	driver.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(1)).WithBlockCreated(time.Now()).Build())
	driver.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(2)).WithBlockCreated(time.Now()).Build())

	driver.blockStorage.HandleBlockAvailabilityRequest(input)

	driver.verifyMocks(t)
}

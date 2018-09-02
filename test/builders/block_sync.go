package builders

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

func BlockAvailabilityResponseInput(lastCommittedBlockHeight primitives.BlockHeight, firstBlockHeight primitives.BlockHeight, lastBlockHeight primitives.BlockHeight, senderPublicKey primitives.Ed25519PublicKey, recipientPublicKey primitives.Ed25519PublicKey) *gossiptopics.BlockAvailabilityResponseInput {
	return &gossiptopics.BlockAvailabilityResponseInput{
		RecipientPublicKey: recipientPublicKey,
		Message: &gossipmessages.BlockAvailabilityResponseMessage{
			SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
				LastCommittedBlockHeight: lastCommittedBlockHeight,
				FirstBlockHeight:         firstBlockHeight,
				LastBlockHeight:          lastBlockHeight,
			}).Build(),
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: senderPublicKey,
			}).Build(),
		},
	}
}

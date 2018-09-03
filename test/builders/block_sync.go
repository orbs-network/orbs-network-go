package builders

import (
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"time"
)

type availabilityResponse struct {
	lastCommittedBlockHeight primitives.BlockHeight
	firstBlockHeight         primitives.BlockHeight
	lastBlockHeight          primitives.BlockHeight
	senderPublicKey          primitives.Ed25519PublicKey
	recipientPublicKey       primitives.Ed25519PublicKey
}

func BlockAvailabilityResponseInput() *availabilityResponse {
	return &availabilityResponse{
		recipientPublicKey:       keys.Ed25519KeyPairForTests(1).PublicKey(),
		senderPublicKey:          keys.Ed25519KeyPairForTests(2).PublicKey(),
		lastBlockHeight:          100,
		lastCommittedBlockHeight: 100,
		firstBlockHeight:         10,
	}
}

func (ar *availabilityResponse) WithSenderPublicKey(publicKey primitives.Ed25519PublicKey) *availabilityResponse {
	ar.senderPublicKey = publicKey
	return ar
}

func (ar *availabilityResponse) WithRecipientPublicKey(publicKey primitives.Ed25519PublicKey) *availabilityResponse {
	ar.recipientPublicKey = publicKey
	return ar
}

func (ar *availabilityResponse) WithLastCommittedBlockHeight(h primitives.BlockHeight) *availabilityResponse {
	ar.lastCommittedBlockHeight = h
	return ar
}

func (ar *availabilityResponse) WithFirstBlockHeight(h primitives.BlockHeight) *availabilityResponse {
	ar.firstBlockHeight = h
	return ar
}

func (ar *availabilityResponse) WithLastBlockHeight(h primitives.BlockHeight) *availabilityResponse {
	ar.lastBlockHeight = h
	return ar
}

func (ar *availabilityResponse) Build() *gossiptopics.BlockAvailabilityResponseInput {
	return &gossiptopics.BlockAvailabilityResponseInput{
		RecipientPublicKey: ar.recipientPublicKey,
		Message: &gossipmessages.BlockAvailabilityResponseMessage{
			SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
				LastCommittedBlockHeight: ar.lastCommittedBlockHeight,
				FirstBlockHeight:         ar.firstBlockHeight,
				LastBlockHeight:          ar.lastBlockHeight,
			}).Build(),
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: ar.senderPublicKey,
			}).Build(),
		},
	}
}

type blockChunk struct {
	blocks []*protocol.BlockPairContainer
	availabilityResponse
}

func BlockSyncResponseInput() *blockChunk {
	chunk := &blockChunk{}
	chunk.availabilityResponse = *BlockAvailabilityResponseInput()

	return chunk
}

func (bc *blockChunk) WithSenderPublicKey(publicKey primitives.Ed25519PublicKey) *blockChunk {
	bc.senderPublicKey = publicKey
	return bc
}

func (bc *blockChunk) WithRecipientPublicKey(publicKey primitives.Ed25519PublicKey) *blockChunk {
	bc.recipientPublicKey = publicKey
	return bc
}

func (bc *blockChunk) WithLastCommittedBlockHeight(h primitives.BlockHeight) *blockChunk {
	bc.lastCommittedBlockHeight = h
	return bc
}

func (bc *blockChunk) WithFirstBlockHeight(h primitives.BlockHeight) *blockChunk {
	bc.firstBlockHeight = h
	return bc
}

func (bc *blockChunk) WithLastBlockHeight(h primitives.BlockHeight) *blockChunk {
	bc.lastBlockHeight = h
	return bc
}

func (bc *blockChunk) Build() *gossiptopics.BlockSyncResponseInput {
	var blocks []*protocol.BlockPairContainer

	for i := bc.firstBlockHeight; i <= bc.lastBlockHeight; i++ {
		blocks = append(blocks, BlockPair().WithHeight(i).WithBlockCreated(time.Now()).Build())
	}

	return &gossiptopics.BlockSyncResponseInput{
		Message: &gossipmessages.BlockSyncResponseMessage{
			SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
				FirstBlockHeight:         bc.firstBlockHeight,
				LastBlockHeight:          bc.lastBlockHeight,
				LastCommittedBlockHeight: bc.lastCommittedBlockHeight,
			}).Build(),
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: bc.senderPublicKey,
			}).Build(),
			BlockPairs: blocks,
		},
	}
}

package builders

import (
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"time"
)

type basicSyncMessage struct {
	lastCommittedBlockHeight primitives.BlockHeight
	firstBlockHeight         primitives.BlockHeight
	lastBlockHeight          primitives.BlockHeight
	senderPublicKey          primitives.Ed25519PublicKey
	recipientPublicKey       primitives.Ed25519PublicKey
}

type availabilityResponse basicSyncMessage

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
	basicSyncMessage
}

func BlockSyncResponseInput() *blockChunk {
	chunk := &blockChunk{}
	chunk.recipientPublicKey = keys.Ed25519KeyPairForTests(1).PublicKey()
	chunk.senderPublicKey = keys.Ed25519KeyPairForTests(2).PublicKey()
	chunk.lastBlockHeight = 100
	chunk.lastCommittedBlockHeight = 100
	chunk.firstBlockHeight = 10

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

type blockAvailabilityRequest basicSyncMessage

func BlockAvailabilityRequestInput() *blockAvailabilityRequest {
	availabilityRequest := &blockAvailabilityRequest{}
	availabilityRequest.recipientPublicKey = keys.Ed25519KeyPairForTests(1).PublicKey()
	availabilityRequest.senderPublicKey = keys.Ed25519KeyPairForTests(2).PublicKey()
	availabilityRequest.lastBlockHeight = 100
	availabilityRequest.lastCommittedBlockHeight = 100
	availabilityRequest.firstBlockHeight = 10

	return availabilityRequest
}

func (bar *blockAvailabilityRequest) WithSenderPublicKey(publicKey primitives.Ed25519PublicKey) *blockAvailabilityRequest {
	bar.senderPublicKey = publicKey
	return bar
}

func (bar *blockAvailabilityRequest) WithRecipientPublicKey(publicKey primitives.Ed25519PublicKey) *blockAvailabilityRequest {
	bar.recipientPublicKey = publicKey
	return bar
}

func (bar *blockAvailabilityRequest) WithLastCommittedBlockHeight(h primitives.BlockHeight) *blockAvailabilityRequest {
	bar.lastCommittedBlockHeight = h
	return bar
}

func (bar *blockAvailabilityRequest) WithFirstBlockHeight(h primitives.BlockHeight) *blockAvailabilityRequest {
	bar.firstBlockHeight = h
	return bar
}

func (bar *blockAvailabilityRequest) WithLastBlockHeight(h primitives.BlockHeight) *blockAvailabilityRequest {
	bar.lastBlockHeight = h
	return bar
}

func (bar *blockAvailabilityRequest) Build() *gossiptopics.BlockAvailabilityRequestInput {
	return &gossiptopics.BlockAvailabilityRequestInput{
		Message: &gossipmessages.BlockAvailabilityRequestMessage{
			SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
				FirstBlockHeight:         bar.firstBlockHeight,
				LastBlockHeight:          bar.lastBlockHeight,
				LastCommittedBlockHeight: bar.lastCommittedBlockHeight,
			}).Build(),
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: bar.senderPublicKey,
			}).Build(),
		},
	}
}

type blockSyncRequest basicSyncMessage

func BlockSyncRequestInput() *blockSyncRequest {
	syncRequest := &blockSyncRequest{}
	syncRequest.recipientPublicKey = keys.Ed25519KeyPairForTests(1).PublicKey()
	syncRequest.senderPublicKey = keys.Ed25519KeyPairForTests(2).PublicKey()
	syncRequest.lastBlockHeight = 100
	syncRequest.lastCommittedBlockHeight = 100
	syncRequest.firstBlockHeight = 10

	return syncRequest
}

func (bsr *blockSyncRequest) WithSenderPublicKey(publicKey primitives.Ed25519PublicKey) *blockSyncRequest {
	bsr.senderPublicKey = publicKey
	return bsr
}

func (bsr *blockSyncRequest) WithRecipientPublicKey(publicKey primitives.Ed25519PublicKey) *blockSyncRequest {
	bsr.recipientPublicKey = publicKey
	return bsr
}

func (bsr *blockSyncRequest) WithLastCommittedBlockHeight(h primitives.BlockHeight) *blockSyncRequest {
	bsr.lastCommittedBlockHeight = h
	return bsr
}

func (bsr *blockSyncRequest) WithFirstBlockHeight(h primitives.BlockHeight) *blockSyncRequest {
	bsr.firstBlockHeight = h
	return bsr
}

func (bsr *blockSyncRequest) WithLastBlockHeight(h primitives.BlockHeight) *blockSyncRequest {
	bsr.lastBlockHeight = h
	return bsr
}

func (bsr *blockSyncRequest) Build() *gossiptopics.BlockSyncRequestInput {
	return &gossiptopics.BlockSyncRequestInput{
		Message: &gossipmessages.BlockSyncRequestMessage{
			SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
				FirstBlockHeight:         bsr.firstBlockHeight,
				LastBlockHeight:          bsr.lastBlockHeight,
				LastCommittedBlockHeight: bsr.lastCommittedBlockHeight,
			}).Build(),
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: bsr.senderPublicKey,
			}).Build(),
		},
	}
}

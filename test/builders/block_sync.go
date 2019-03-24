// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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
	senderNodeAddress        primitives.NodeAddress
	recipientNodeAddress     primitives.NodeAddress
}

type availabilityResponse basicSyncMessage

func BlockAvailabilityResponseInput() *availabilityResponse {
	return &availabilityResponse{
		recipientNodeAddress:     keys.EcdsaSecp256K1KeyPairForTests(1).NodeAddress(),
		senderNodeAddress:        keys.EcdsaSecp256K1KeyPairForTests(2).NodeAddress(),
		lastBlockHeight:          100,
		lastCommittedBlockHeight: 100,
		firstBlockHeight:         10,
	}
}

func (ar *availabilityResponse) WithSenderNodeAddress(nodeAddress primitives.NodeAddress) *availabilityResponse {
	ar.senderNodeAddress = nodeAddress
	return ar
}

func (ar *availabilityResponse) WithRecipientNodeAddress(nodeAddress primitives.NodeAddress) *availabilityResponse {
	ar.recipientNodeAddress = nodeAddress
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
		RecipientNodeAddress: ar.recipientNodeAddress,
		Message: &gossipmessages.BlockAvailabilityResponseMessage{
			SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
				LastCommittedBlockHeight: ar.lastCommittedBlockHeight,
				FirstBlockHeight:         ar.firstBlockHeight,
				LastBlockHeight:          ar.lastBlockHeight,
			}).Build(),
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderNodeAddress: ar.senderNodeAddress,
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
	chunk.recipientNodeAddress = keys.EcdsaSecp256K1KeyPairForTests(1).NodeAddress()
	chunk.senderNodeAddress = keys.EcdsaSecp256K1KeyPairForTests(2).NodeAddress()
	chunk.lastBlockHeight = 100
	chunk.lastCommittedBlockHeight = 100
	chunk.firstBlockHeight = 10

	return chunk
}

func (bc *blockChunk) WithSenderNodeAddress(nodeAddress primitives.NodeAddress) *blockChunk {
	bc.senderNodeAddress = nodeAddress
	return bc
}

func (bc *blockChunk) WithRecipientNodeAddress(nodeAddress primitives.NodeAddress) *blockChunk {
	bc.recipientNodeAddress = nodeAddress
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
		blockTime := time.Unix(1550394190000000000+int64(i), 0) // deterministic block creation in the past based on block height
		blocks = append(blocks, BlockPair().WithHeight(i).WithBlockCreated(blockTime).Build())
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
				SenderNodeAddress: bc.senderNodeAddress,
			}).Build(),
			BlockPairs: blocks,
		},
	}
}

type blockAvailabilityRequest basicSyncMessage

func BlockAvailabilityRequestInput() *blockAvailabilityRequest {
	availabilityRequest := &blockAvailabilityRequest{}
	availabilityRequest.recipientNodeAddress = keys.EcdsaSecp256K1KeyPairForTests(1).NodeAddress()
	availabilityRequest.senderNodeAddress = keys.EcdsaSecp256K1KeyPairForTests(2).NodeAddress()
	availabilityRequest.lastBlockHeight = 100
	availabilityRequest.lastCommittedBlockHeight = 100
	availabilityRequest.firstBlockHeight = 10

	return availabilityRequest
}

func (bar *blockAvailabilityRequest) WithSenderNodeAddress(nodeAddress primitives.NodeAddress) *blockAvailabilityRequest {
	bar.senderNodeAddress = nodeAddress
	return bar
}

func (bar *blockAvailabilityRequest) WithRecipientNodeAddress(nodeAddress primitives.NodeAddress) *blockAvailabilityRequest {
	bar.recipientNodeAddress = nodeAddress
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
				SenderNodeAddress: bar.senderNodeAddress,
			}).Build(),
		},
	}
}

type blockSyncRequest basicSyncMessage

func BlockSyncRequestInput() *blockSyncRequest {
	syncRequest := &blockSyncRequest{}
	syncRequest.recipientNodeAddress = keys.EcdsaSecp256K1KeyPairForTests(1).NodeAddress()
	syncRequest.senderNodeAddress = keys.EcdsaSecp256K1KeyPairForTests(2).NodeAddress()
	syncRequest.lastBlockHeight = 100
	syncRequest.lastCommittedBlockHeight = 100
	syncRequest.firstBlockHeight = 10

	return syncRequest
}

func (bsr *blockSyncRequest) WithSenderNodeAddress(nodeAddress primitives.NodeAddress) *blockSyncRequest {
	bsr.senderNodeAddress = nodeAddress
	return bsr
}

func (bsr *blockSyncRequest) WithRecipientNodeAddress(nodeAddress primitives.NodeAddress) *blockSyncRequest {
	bsr.recipientNodeAddress = nodeAddress
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
				SenderNodeAddress: bsr.senderNodeAddress,
			}).Build(),
		},
	}
}

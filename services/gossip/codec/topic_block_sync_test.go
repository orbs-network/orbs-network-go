// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package codec

import (
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBlockSync_BlockAvailabilityRequest(t *testing.T) {
	message := &gossipmessages.BlockAvailabilityRequestMessage{
		SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
			BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: []byte{0x01, 0x02, 0x03},
			Signature:         []byte{0x04, 0x05, 0x06},
		}).Build(),
	}

	payloads, err := EncodeBlockAvailabilityRequest((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBlockAvailabilityRequest(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
}

func TestBlockSync_EmptyBlockAvailabilityRequest(t *testing.T) {
	_, err := DecodeBlockAvailabilityRequest(builders.EmptyPayloads(2))
	require.Error(t, err, "decode should fail and return error")
}

func TestBlockSync_BlockAvailabilityRequestDoNotFailWhenSenderContainsNil(t *testing.T) {
	message := &gossipmessages.BlockAvailabilityRequestMessage{
		SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
			BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: nil,
			Signature:         nil,
		}).Build(),
	}

	payloads, err := EncodeBlockAvailabilityRequest((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBlockAvailabilityRequest(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
	test.RequireDoesNotContainNil(t, decoded)
}

func TestBlockSync_BlockAvailabilityResponse(t *testing.T) {
	message := &gossipmessages.BlockAvailabilityResponseMessage{
		SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
			BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: []byte{0x01, 0x02, 0x03},
			Signature:         []byte{0x04, 0x05, 0x06},
		}).Build(),
	}

	payloads, err := EncodeBlockAvailabilityResponse((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBlockAvailabilityResponse(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
}

func TestBlockSync_EmptyBlockAvailabilityResponse(t *testing.T) {
	_, err := DecodeBlockAvailabilityResponse(builders.EmptyPayloads(2))
	require.Error(t, err, "decode should fail and return error")
}

func TestBlockSync_BlockAvailabilityResponseDoNotFailWhenSenderContainsNil(t *testing.T) {
	message := &gossipmessages.BlockAvailabilityResponseMessage{
		SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
			BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: nil,
			Signature:         nil,
		}).Build(),
	}

	payloads, err := EncodeBlockAvailabilityResponse((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBlockAvailabilityResponse(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
	test.RequireDoesNotContainNil(t, decoded)
}

func TestBlockSync_BlockSyncRequest(t *testing.T) {
	message := &gossipmessages.BlockSyncRequestMessage{
		SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
			BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: []byte{0x01, 0x02, 0x03},
			Signature:         []byte{0x04, 0x05, 0x06},
		}).Build(),
	}

	payloads, err := EncodeBlockSyncRequest((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBlockSyncRequest(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
}

func TestBlockSync_EmptyBlockSyncRequest(t *testing.T) {
	_, err := DecodeBlockSyncRequest(builders.EmptyPayloads(2))
	require.Error(t, err, "decode should fail and return error")
}

func TestBlockSync_BlockSyncRequestDoNotFailWhenSenderContainsNil(t *testing.T) {
	message := &gossipmessages.BlockSyncRequestMessage{
		SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
			BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: nil,
			Signature:         nil,
		}).Build(),
	}

	payloads, err := EncodeBlockSyncRequest((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBlockSyncRequest(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
	test.RequireDoesNotContainNil(t, decoded)
}

func TestBlockSync_BlockSyncResponse(t *testing.T) {
	message := &gossipmessages.BlockSyncResponseMessage{
		SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
			BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: []byte{0x01, 0x02, 0x03},
			Signature:         []byte{0x04, 0x05, 0x06},
		}).Build(),
		BlockPairs: []*protocol.BlockPairContainer{
			builders.BlockPair().WithTransactions(5).Build(),
			builders.BlockPair().WithTransactions(3).Build(),
		},
	}

	payloads, err := EncodeBlockSyncResponse((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBlockSyncResponse(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
}

func TestBlockSync_EmptyBlockSyncResponse(t *testing.T) {
	_, err := DecodeBlockSyncResponse(builders.EmptyPayloads(2 + NUM_HARDCODED_PAYLOADS_FOR_BLOCK_PAIR))
	require.Error(t, err, "decode should fail and return error")
}

func TestBlockSync_BlockSyncResponseDoNotFailWhenSenderContainsNil(t *testing.T) {
	message := &gossipmessages.BlockSyncResponseMessage{
		SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
			BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: nil,
			Signature:         nil,
		}).Build(),
		BlockPairs: []*protocol.BlockPairContainer{
			builders.BlockPair().WithTransactions(5).Build(),
			builders.BlockPair().WithTransactions(3).Build(),
		},
	}

	payloads, err := EncodeBlockSyncResponse((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBlockSyncResponse(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
	test.RequireDoesNotContainNil(t, decoded)
}

func TestBlockSync_BlockSyncResponseWithCorruptNumTransactions(t *testing.T) {
	message := &gossipmessages.BlockSyncResponseMessage{
		SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
			BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: []byte{0x01, 0x02, 0x03},
			Signature:         []byte{0x04, 0x05, 0x06},
		}).Build(),
		BlockPairs: []*protocol.BlockPairContainer{
			builders.BlockPair().WithCorruptNumTransactions(3).Build(),
			builders.BlockPair().WithCorruptNumTransactions(2).Build(),
		},
	}

	payloads, err := EncodeBlockSyncResponse((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	_, err = DecodeBlockSyncResponse(payloads[1:])
	require.Error(t, err, "decode should fail and return error")
}

func TestBlockSync_BlockSyncResponseWithCorruptBlockPair(t *testing.T) {
	message := &gossipmessages.BlockSyncResponseMessage{
		SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
			BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: []byte{0x01, 0x02, 0x03},
			Signature:         []byte{0x04, 0x05, 0x06},
		}).Build(),
		BlockPairs: []*protocol.BlockPairContainer{
			builders.CorruptBlockPair().Build(),
			builders.CorruptBlockPair().Build(),
		},
	}

	_, err := EncodeBlockSyncResponse((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.Error(t, err, "encode should fail and return error")
}

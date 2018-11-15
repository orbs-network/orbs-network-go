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
			SenderPublicKey: []byte{0x01, 0x02, 0x03},
			Signature:       []byte{0x04, 0x05, 0x06},
		}).Build(),
	}

	payloads, err := EncodeBlockAvailabilityRequest((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBlockAvailabilityRequest(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
}

func TestBlockSync_EmptyBlockAvailabilityRequest(t *testing.T) {
	decoded, err := DecodeBlockAvailabilityRequest(emptyPayloads(2))
	require.NoError(t, err, "decode should not fail")
	require.False(t, containsNil(decoded), "decoded should not contain nil fields")
}

func TestBlockSync_BlockAvailabilityRequestDoNotFailWhenPublicKeyIsNil(t *testing.T) {
	message := &gossipmessages.BlockAvailabilityRequestMessage{
		SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
			BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderPublicKey: nil,
			Signature:       []byte{0x04, 0x05, 0x06},
		}).Build(),
	}

	payloads, err := EncodeBlockAvailabilityRequest((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBlockAvailabilityRequest(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	require.False(t, containsNil(decoded), "decoded should not contain nil fields")
}

func TestBlockSync_BlockAvailabilityRequestDoNotFailWhenSignatureIsNil(t *testing.T) {
	message := &gossipmessages.BlockAvailabilityRequestMessage{
		SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
			BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderPublicKey: []byte{0x01, 0x02, 0x03},
			Signature:       nil,
		}).Build(),
	}

	payloads, err := EncodeBlockAvailabilityRequest((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBlockAvailabilityRequest(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	require.False(t, containsNil(decoded), "decoded should not contain nil fields")
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
			SenderPublicKey: []byte{0x01, 0x02, 0x03},
			Signature:       []byte{0x04, 0x05, 0x06},
		}).Build(),
	}

	payloads, err := EncodeBlockAvailabilityResponse((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBlockAvailabilityResponse(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
}

func TestBlockSync_EmptyBlockAvailabilityResponse(t *testing.T) {
	decoded, err := DecodeBlockAvailabilityResponse(emptyPayloads(2))
	require.NoError(t, err, "decode should not fail")
	require.False(t, containsNil(decoded), "decoded should not contain nil fields")
}

func TestBlockSync_BlockAvailabilityResponseDoNotFailWhenPublicKeyIsNil(t *testing.T) {
	message := &gossipmessages.BlockAvailabilityResponseMessage{
		SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
			BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderPublicKey: nil,
			Signature:       []byte{0x04, 0x05, 0x06},
		}).Build(),
	}

	payloads, err := EncodeBlockAvailabilityResponse((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBlockAvailabilityResponse(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	require.False(t, containsNil(decoded), "decoded should not contain nil fields")
}

func TestBlockSync_BlockAvailabilityResponseDoNotFailWhenSignatureIsNil(t *testing.T) {
	message := &gossipmessages.BlockAvailabilityResponseMessage{
		SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
			BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderPublicKey: []byte{0x01, 0x02, 0x03},
			Signature:       nil,
		}).Build(),
	}

	payloads, err := EncodeBlockAvailabilityResponse((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBlockAvailabilityResponse(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	require.False(t, containsNil(decoded), "decoded should not contain nil fields")
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
			SenderPublicKey: []byte{0x01, 0x02, 0x03},
			Signature:       []byte{0x04, 0x05, 0x06},
		}).Build(),
	}

	payloads, err := EncodeBlockSyncRequest((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBlockSyncRequest(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
}

func TestBlockSync_EmptyBlockSyncRequest(t *testing.T) {
	decoded, err := DecodeBlockSyncRequest(emptyPayloads(2))
	require.NoError(t, err, "decode should not fail")
	require.False(t, containsNil(decoded), "decoded should not contain nil fields")
}

func TestBlockSync_BlockSyncRequestDoNotFailWhenPublicKeyIsNil(t *testing.T) {
	message := &gossipmessages.BlockSyncRequestMessage{
		SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
			BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderPublicKey: nil,
			Signature:       []byte{0x04, 0x05, 0x06},
		}).Build(),
	}

	payloads, err := EncodeBlockSyncRequest((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBlockSyncRequest(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	require.False(t, containsNil(decoded), "decoded should not contain nil fields")
}

func TestBlockSync_BlockSyncRequestDoNotFailWhenSignatureIsNil(t *testing.T) {
	message := &gossipmessages.BlockSyncRequestMessage{
		SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
			BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderPublicKey: []byte{0x01, 0x02, 0x03},
			Signature:       nil,
		}).Build(),
	}

	payloads, err := EncodeBlockSyncRequest((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBlockSyncRequest(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	require.False(t, containsNil(decoded), "decoded should not contain nil fields")
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
			SenderPublicKey: []byte{0x01, 0x02, 0x03},
			Signature:       []byte{0x04, 0x05, 0x06},
		}).Build(),
		BlockPairs: []*protocol.BlockPairContainer{
			builders.BlockPair().WithTransactions(5).WithReceipts(5).WithStateDiffs(3).Build(),
			builders.BlockPair().WithTransactions(3).WithReceipts(3).WithStateDiffs(2).Build(),
		},
	}

	payloads, err := EncodeBlockSyncResponse((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBlockSyncResponse(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
}

// TODO: add more tests for blocksyncresponse

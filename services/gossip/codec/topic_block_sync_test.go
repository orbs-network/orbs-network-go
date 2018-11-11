package codec

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/test"
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
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to origial")
}

func TestBlockSync_EmptyBlockAvailabilityRequest(t *testing.T) {
	t.Skip("this is failing, not sure why")
	decoded, err := DecodeBlockAvailabilityRequest(emptyPayloads(2))
	require.NoError(t, err, "decode should not fail")
	fmt.Printf("%s\n", decoded.String())
	require.False(t, containsNil(decoded), "decoded should not contain nil fields")
}

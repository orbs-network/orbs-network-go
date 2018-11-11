package codec

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/pkg/errors"
)

func EncodeBlockAvailabilityRequest(message *gossipmessages.BlockAvailabilityRequestMessage) ([][]byte, error) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:         gossipmessages.HEADER_TOPIC_BLOCK_SYNC,
		BlockSync:     gossipmessages.BLOCK_SYNC_AVAILABILITY_REQUEST,
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
	}).Build()

	if message.SignedBatchRange == nil {
		return nil, errors.New("missing SignedBatchRange")
	}
	return [][]byte{header.Raw(), message.SignedBatchRange.Raw(), message.Sender.Raw()}, nil
}

func DecodeBlockAvailabilityRequest(payloads [][]byte) (*gossipmessages.BlockAvailabilityRequestMessage, error) {
	if len(payloads) < 2 {
		return nil, errors.New("not enough payloads")
	}
	batchRange := gossipmessages.BlockSyncRangeReader(payloads[0])
	senderSignature := gossipmessages.SenderSignatureReader(payloads[1])
	if !senderSignature.IsValid() {
		return nil, errors.New("senderSignature is not valid")
	}
	return &gossipmessages.BlockAvailabilityRequestMessage{
		SignedBatchRange: batchRange,
		Sender:           senderSignature,
	}, nil
}

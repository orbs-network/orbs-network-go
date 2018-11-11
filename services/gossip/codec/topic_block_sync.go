package codec

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/pkg/errors"
)

func EncodeBlockAvailabilityRequest(header *gossipmessages.Header, message *gossipmessages.BlockAvailabilityRequestMessage) ([][]byte, error) {
	if message.SignedBatchRange == nil {
		return nil, errors.New("missing SignedBatchRange")
	}
	return [][]byte{header.Raw(), message.SignedBatchRange.Raw(), message.Sender.Raw()}, nil
}

func DecodeBlockAvailabilityRequest(payloads [][]byte) (*gossipmessages.BlockAvailabilityRequestMessage, error) {
	if len(payloads) != 2 {
		return nil, errors.New("wrong num of payloads")
	}
	batchRange := gossipmessages.BlockSyncRangeReader(payloads[0])
	senderSignature := gossipmessages.SenderSignatureReader(payloads[1])
	return &gossipmessages.BlockAvailabilityRequestMessage{
		SignedBatchRange: batchRange,
		Sender:           senderSignature,
	}, nil
}

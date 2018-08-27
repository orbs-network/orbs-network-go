package gossip

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
)

func (s *service) RegisterBlockSyncHandler(handler gossiptopics.BlockSyncHandler) {
	s.blockSyncHandlers = append(s.blockSyncHandlers, handler)
}

func (s *service) receivedBlockSyncMessage(header *gossipmessages.Header, payloads [][]byte) {
	switch header.BlockSync() {
	case gossipmessages.BLOCK_SYNC_AVAILABILITY_REQUEST:
		s.receivedBlockSyncAvailabilityRequest(header, payloads)
	case gossipmessages.BLOCK_SYNC_AVAILABILITY_RESPONSE:
		s.receivedBlockSyncAvailabilityResponse(header, payloads)
	}
}

func (s *service) BroadcastBlockAvailabilityRequest(input *gossiptopics.BlockAvailabilityRequestInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:         gossipmessages.HEADER_TOPIC_BLOCK_SYNC,
		BlockSync:     gossipmessages.BLOCK_SYNC_AVAILABILITY_REQUEST,
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
	}).Build()

	if input.Message.SignedBatchRange == nil {
		return nil, errors.Errorf("cannot encode BlockAvailabilityRequestMessage", log.Stringable("message", input.Message))
	}
	payloads := [][]byte{header.Raw(), input.Message.SignedBatchRange.Raw(), input.Message.Sender.Raw()}

	return nil, s.transport.Send(&adapter.TransportData{
		SenderPublicKey: s.config.NodePublicKey(),
		RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		Payloads:        payloads,
	})
}

func (s *service) receivedBlockSyncAvailabilityRequest(header *gossipmessages.Header, payloads [][]byte) {
	if len(payloads) < 2 {
		return
	}
	batchRange := gossipmessages.BlockSyncRangeReader(payloads[0])
	senderSignature := gossipmessages.SenderSignatureReader(payloads[1])

	for _, l := range s.blockSyncHandlers {
		l.HandleBlockAvailabilityRequest(&gossiptopics.BlockAvailabilityRequestInput{
			Message: &gossipmessages.BlockAvailabilityRequestMessage{
				SignedBatchRange: batchRange,
				Sender:           senderSignature,
			},
		})
	}
}

func (s *service) SendBlockAvailabilityResponse(input *gossiptopics.BlockAvailabilityResponseInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:         gossipmessages.HEADER_TOPIC_BLOCK_SYNC,
		BlockSync:     gossipmessages.BLOCK_SYNC_AVAILABILITY_RESPONSE,
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
	}).Build()

	if input.Message.SignedBatchRange == nil {
		return nil, errors.Errorf("cannot encode BlockAvailabilityResponseMessage", log.Stringable("message", input.Message))
	}
	payloads := [][]byte{header.Raw(), input.Message.SignedBatchRange.Raw(), input.Message.Sender.Raw()}

	return nil, s.transport.Send(&adapter.TransportData{
		SenderPublicKey: s.config.NodePublicKey(),
		RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		Payloads:        payloads,
	})
}

func (s *service) receivedBlockSyncAvailabilityResponse(header *gossipmessages.Header, payloads [][]byte) {
	if len(payloads) < 2 {
		return
	}
	batchRange := gossipmessages.BlockSyncRangeReader(payloads[0])
	senderSignature := gossipmessages.SenderSignatureReader(payloads[1])

	for _, l := range s.blockSyncHandlers {
		l.HandleBlockAvailabilityResponse(&gossiptopics.BlockAvailabilityResponseInput{
			Message: &gossipmessages.BlockAvailabilityResponseMessage{
				SignedBatchRange: batchRange,
				Sender:           senderSignature,
			},
		})
	}
}

func (s *service) SendBlockSyncRequest(input *gossiptopics.BlockSyncRequestInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}
func (s *service) SendBlockSyncResponse(input *gossiptopics.BlockSyncResponseInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
	// TODO this is for Tal
}

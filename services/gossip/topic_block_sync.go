package gossip

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
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
	case gossipmessages.BLOCK_SYNC_REQUEST:
		s.receivedBlockSyncRequest(header, payloads)
	case gossipmessages.BLOCK_SYNC_RESPONSE:
		s.receivedBlockSyncResponse(header, payloads)
	}
}

func (s *service) BroadcastBlockAvailabilityRequest(ctx context.Context, input *gossiptopics.BlockAvailabilityRequestInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:         gossipmessages.HEADER_TOPIC_BLOCK_SYNC,
		BlockSync:     gossipmessages.BLOCK_SYNC_AVAILABILITY_REQUEST,
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
	}).Build()

	if input.Message.SignedBatchRange == nil {
		return nil, errors.Errorf("cannot encode BlockAvailabilityRequestMessage: %s", input.Message.String())
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
		_, err := l.HandleBlockAvailabilityRequest(&gossiptopics.BlockAvailabilityRequestInput{
			Message: &gossipmessages.BlockAvailabilityRequestMessage{
				SignedBatchRange: batchRange,
				Sender:           senderSignature,
			},
		})
		if err != nil {
			s.logger.Info("HandleBlockAvailabilityRequest failed", log.Error(err))
		}
	}
}

func (s *service) SendBlockAvailabilityResponse(ctx context.Context, input *gossiptopics.BlockAvailabilityResponseInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:         gossipmessages.HEADER_TOPIC_BLOCK_SYNC,
		BlockSync:     gossipmessages.BLOCK_SYNC_AVAILABILITY_RESPONSE,
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_LIST,
	}).Build()

	if input.Message.SignedBatchRange == nil {
		return nil, errors.Errorf("cannot encode BlockAvailabilityResponseMessage: %s", input.Message.String())
	}
	payloads := [][]byte{header.Raw(), input.Message.SignedBatchRange.Raw(), input.Message.Sender.Raw()}

	return nil, s.transport.Send(&adapter.TransportData{
		SenderPublicKey: s.config.NodePublicKey(),
		RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_LIST,
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
		_, err := l.HandleBlockAvailabilityResponse(&gossiptopics.BlockAvailabilityResponseInput{
			Message: &gossipmessages.BlockAvailabilityResponseMessage{
				SignedBatchRange: batchRange,
				Sender:           senderSignature,
			},
		})
		if err != nil {
			s.logger.Info("HandleBlockAvailabilityResponse failed", log.Error(err))
		}
	}
}

func (s *service) SendBlockSyncRequest(ctx context.Context, input *gossiptopics.BlockSyncRequestInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:               gossipmessages.HEADER_TOPIC_BLOCK_SYNC,
		BlockSync:           gossipmessages.BLOCK_SYNC_REQUEST,
		RecipientMode:       gossipmessages.RECIPIENT_LIST_MODE_LIST,
		RecipientPublicKeys: []primitives.Ed25519PublicKey{input.RecipientPublicKey},
	}).Build()

	if input.Message.SignedChunkRange == nil {
		return nil, errors.Errorf("cannot encode BlockSyncRequestMessage: %s", input.Message.String())
	}
	payloads := [][]byte{header.Raw(), input.Message.SignedChunkRange.Raw(), input.Message.Sender.Raw()}

	return nil, s.transport.Send(&adapter.TransportData{
		SenderPublicKey:     s.config.NodePublicKey(),
		RecipientMode:       gossipmessages.RECIPIENT_LIST_MODE_LIST,
		RecipientPublicKeys: []primitives.Ed25519PublicKey{input.RecipientPublicKey},
		Payloads:            payloads,
	})
}

func (s *service) receivedBlockSyncRequest(header *gossipmessages.Header, payloads [][]byte) {
	if len(payloads) < 2 {
		return
	}
	chunkRange := gossipmessages.BlockSyncRangeReader(payloads[0])
	senderSignature := gossipmessages.SenderSignatureReader(payloads[1])

	for _, l := range s.blockSyncHandlers {
		_, err := l.HandleBlockSyncRequest(&gossiptopics.BlockSyncRequestInput{
			Message: &gossipmessages.BlockSyncRequestMessage{
				SignedChunkRange: chunkRange,
				Sender:           senderSignature,
			},
		})
		if err != nil {
			s.logger.Info("HandleBlockSyncRequest failed", log.Error(err))
		}
	}
}

func (s *service) SendBlockSyncResponse(ctx context.Context, input *gossiptopics.BlockSyncResponseInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:               gossipmessages.HEADER_TOPIC_BLOCK_SYNC,
		BlockSync:           gossipmessages.BLOCK_SYNC_RESPONSE,
		RecipientMode:       gossipmessages.RECIPIENT_LIST_MODE_LIST,
		RecipientPublicKeys: []primitives.Ed25519PublicKey{input.RecipientPublicKey},
	}).Build()

	if input.Message.SignedChunkRange == nil || len(input.Message.BlockPairs) == 0 {
		return nil, errors.Errorf("cannot encode BlockSyncResponseMessage: %s", input.Message.String())
	}
	payloads := [][]byte{header.Raw(), input.Message.SignedChunkRange.Raw(), input.Message.Sender.Raw()}

	blockPairPayloads, err := encodeBlockPairs(input.Message.BlockPairs)
	if err != nil {
		return nil, err
	}
	payloads = append(payloads, blockPairPayloads...)

	return nil, s.transport.Send(&adapter.TransportData{
		SenderPublicKey:     s.config.NodePublicKey(),
		RecipientMode:       gossipmessages.RECIPIENT_LIST_MODE_LIST,
		RecipientPublicKeys: []primitives.Ed25519PublicKey{input.RecipientPublicKey},
		Payloads:            payloads,
	})
}

func (s *service) receivedBlockSyncResponse(header *gossipmessages.Header, payloads [][]byte) {
	if len(payloads) < 3 {
		return
	}
	chunkRange := gossipmessages.BlockSyncRangeReader(payloads[0])
	senderSignature := gossipmessages.SenderSignatureReader(payloads[1])

	blocks, err := decodeBlockPairs(payloads)

	if err != nil {
		s.logger.Error("could not decode block pair from block sync", log.Error(err))
		return
	}

	for _, l := range s.blockSyncHandlers {
		_, err := l.HandleBlockSyncResponse(&gossiptopics.BlockSyncResponseInput{
			Message: &gossipmessages.BlockSyncResponseMessage{
				SignedChunkRange: chunkRange,
				Sender:           senderSignature,
				BlockPairs:       blocks,
			},
		})
		if err != nil {
			s.logger.Info("HandleBlockSyncResponse failed", log.Error(err))
		}
	}
}

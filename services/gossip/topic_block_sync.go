package gossip

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/gossip/codec"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
)

func (s *service) RegisterBlockSyncHandler(handler gossiptopics.BlockSyncHandler) {
	s.blockSyncHandlers = append(s.blockSyncHandlers, handler)
}

func (s *service) receivedBlockSyncMessage(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
	switch header.BlockSync() {
	case gossipmessages.BLOCK_SYNC_AVAILABILITY_REQUEST:
		s.receivedBlockSyncAvailabilityRequest(ctx, header, payloads)
	case gossipmessages.BLOCK_SYNC_AVAILABILITY_RESPONSE:
		s.receivedBlockSyncAvailabilityResponse(ctx, header, payloads)
	case gossipmessages.BLOCK_SYNC_REQUEST:
		s.receivedBlockSyncRequest(ctx, header, payloads)
	case gossipmessages.BLOCK_SYNC_RESPONSE:
		s.receivedBlockSyncResponse(ctx, header, payloads)
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

	return nil, s.transport.Send(ctx, &adapter.TransportData{
		SenderPublicKey: s.config.NodePublicKey(),
		RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		Payloads:        payloads,
	})
}

func (s *service) receivedBlockSyncAvailabilityRequest(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
	if len(payloads) < 2 {
		return
	}
	batchRange := gossipmessages.BlockSyncRangeReader(payloads[0])
	senderSignature := gossipmessages.SenderSignatureReader(payloads[1])
	// attempting with talkol to fix issue #437
	if !senderSignature.IsValid() {
		return
	}

	for _, l := range s.blockSyncHandlers {
		_, err := l.HandleBlockAvailabilityRequest(ctx, &gossiptopics.BlockAvailabilityRequestInput{
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
		Topic:               gossipmessages.HEADER_TOPIC_BLOCK_SYNC,
		BlockSync:           gossipmessages.BLOCK_SYNC_AVAILABILITY_RESPONSE,
		RecipientMode:       gossipmessages.RECIPIENT_LIST_MODE_LIST,
		RecipientPublicKeys: []primitives.Ed25519PublicKey{input.RecipientPublicKey},
	}).Build()

	if input.Message.SignedBatchRange == nil {
		return nil, errors.Errorf("cannot encode BlockAvailabilityResponseMessage: %s", input.Message.String())
	}
	payloads := [][]byte{header.Raw(), input.Message.SignedBatchRange.Raw(), input.Message.Sender.Raw()}

	return nil, s.transport.Send(ctx, &adapter.TransportData{
		SenderPublicKey:     s.config.NodePublicKey(),
		RecipientMode:       gossipmessages.RECIPIENT_LIST_MODE_LIST,
		RecipientPublicKeys: []primitives.Ed25519PublicKey{input.RecipientPublicKey},
		Payloads:            payloads,
	})
}

func (s *service) receivedBlockSyncAvailabilityResponse(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
	if len(payloads) < 2 {
		return
	}
	batchRange := gossipmessages.BlockSyncRangeReader(payloads[0])
	senderSignature := gossipmessages.SenderSignatureReader(payloads[1])

	for _, l := range s.blockSyncHandlers {
		_, err := l.HandleBlockAvailabilityResponse(ctx, &gossiptopics.BlockAvailabilityResponseInput{
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

	return nil, s.transport.Send(ctx, &adapter.TransportData{
		SenderPublicKey:     s.config.NodePublicKey(),
		RecipientMode:       gossipmessages.RECIPIENT_LIST_MODE_LIST,
		RecipientPublicKeys: []primitives.Ed25519PublicKey{input.RecipientPublicKey},
		Payloads:            payloads,
	})
}

func (s *service) receivedBlockSyncRequest(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
	if len(payloads) < 2 {
		return
	}
	chunkRange := gossipmessages.BlockSyncRangeReader(payloads[0])
	senderSignature := gossipmessages.SenderSignatureReader(payloads[1])

	for _, l := range s.blockSyncHandlers {
		_, err := l.HandleBlockSyncRequest(ctx, &gossiptopics.BlockSyncRequestInput{
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

	blockPairPayloads, err := codec.EncodeBlockPairs(input.Message.BlockPairs)
	if err != nil {
		return nil, err
	}
	payloads = append(payloads, blockPairPayloads...)

	return nil, s.transport.Send(ctx, &adapter.TransportData{
		SenderPublicKey:     s.config.NodePublicKey(),
		RecipientMode:       gossipmessages.RECIPIENT_LIST_MODE_LIST,
		RecipientPublicKeys: []primitives.Ed25519PublicKey{input.RecipientPublicKey},
		Payloads:            payloads,
	})
}

func (s *service) receivedBlockSyncResponse(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
	if len(payloads) < 2 {
		return
	}
	chunkRange := gossipmessages.BlockSyncRangeReader(payloads[0])
	senderSignature := gossipmessages.SenderSignatureReader(payloads[1])

	blocks, err := codec.DecodeBlockPairs(payloads[2:])

	if err != nil {
		s.logger.Error("could not decode block pair from block sync", log.Error(err))
		return
	}

	for _, l := range s.blockSyncHandlers {
		_, err := l.HandleBlockSyncResponse(ctx, &gossiptopics.BlockSyncResponseInput{
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

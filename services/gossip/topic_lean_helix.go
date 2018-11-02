package gossip

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
)

func (s *service) RegisterLeanHelixHandler(handler gossiptopics.LeanHelixHandler) {
	s.leanHelixHandlers = append(s.leanHelixHandlers, handler)
}

func (s *service) receivedLeanHelixMessage(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {

	if len(payloads) < 2 {
		return
	}

	messageType := header.LeanHelix()
	content := payloads[1]
	blockPair, err := decodeBlockPair(payloads[2:])
	if err != nil {
		return
	}

	for _, l := range s.leanHelixHandlers {
		_, err := l.HandleLeanHelixMessage(ctx, &gossiptopics.LeanHelixInput{
			Message: &gossipmessages.LeanHelixMessage{
				MessageType: messageType,
				Content:     content,
				BlockPair:   blockPair,
			},
		})
		if err != nil {
			s.logger.Info("HandleLeanHelixMessage failed", log.Error(err))
		}
	}
}

func (s *service) SendLeanHelixMessage(ctx context.Context, input *gossiptopics.LeanHelixInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:         gossipmessages.HEADER_TOPIC_LEAN_HELIX,
		LeanHelix:     input.Message.MessageType,
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
	}).Build()

	blockPairPayloads, err := encodeBlockPair(input.Message.BlockPair)
	if err != nil {
		return nil, err
	}
	if input.Message.Content == nil {
		return nil, errors.Errorf("cannot encode LeanHelixMessage: %s", input.Message.String())
	}
	payloads := append([][]byte{header.Raw(), input.Message.Content}, blockPairPayloads...)

	return nil, s.transport.Send(ctx, &adapter.TransportData{
		SenderPublicKey: s.config.NodePublicKey(),
		RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_BROADCAST, // TODO: shouldn't be broadcast
		Payloads:        payloads,
	})
}

//func (s *service) receivedLeanHelixPrePrepare(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
//	if len(payloads) < 2 {
//		return
//	}
//	signedHeader := consensus.LeanHelixBlockRefReader(payloads[0])
//	senderSignature := consensus.LeanHelixSenderSignatureReader(payloads[1])
//	blockPair, err := decodeBlockPair(payloads[2:])
//	if err != nil {
//		return
//	}
//
//	for _, l := range s.leanHelixHandlers {
//		_, err := l.HandleLeanHelixPrePrepare(ctx, &gossiptopics.LeanHelixPrePrepareInput{
//			Message: &gossipmessages.LeanHelixPrePrepareMessage{
//				SignedHeader: signedHeader,
//				Sender:       senderSignature,
//				BlockPair:    blockPair,
//			},
//		})
//		if err != nil {
//			s.logger.Info("HandleLeanHelixPrePrepare failed", log.Error(err))
//		}
//	}
//}

//func (s *service) SendLeanHelixPrepare(ctx context.Context, input *gossiptopics.LeanHelixPrepareInput) (*gossiptopics.EmptyOutput, error) {
//	header := (&gossipmessages.HeaderBuilder{
//		Topic:         gossipmessages.HEADER_TOPIC_LEAN_HELIX,
//		LeanHelix:     consensus.LEAN_HELIX_PREPARE,
//		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
//	}).Build()
//
//	payloads := [][]byte{header.Raw()}
//
//	return nil, s.transport.Send(ctx, &adapter.TransportData{
//		SenderPublicKey: s.config.NodePublicKey(),
//		RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_BROADCAST, // TODO: shouldn't be broadcast
//		Payloads:        payloads,
//	})
//}

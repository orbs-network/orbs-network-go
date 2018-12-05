package gossip

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
)

func (s *service) RegisterLeanHelixHandler(handler gossiptopics.LeanHelixHandler) {
	s.leanHelixHandlers = append(s.leanHelixHandlers, handler)
}

func (s *service) receivedLeanHelixMessage(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {

	messageType := header.LeanHelix()

	if len(payloads) < 1 {
		s.logger.Info("receivedLeanHelixMessage() too little payloads!")
		return
	}

	var blockPair *protocol.BlockPairContainer

	content := payloads[0]
	if len(payloads) > 1 {
		var err error
		blockPair, err = decodeBlockPair(payloads[1:])
		if err != nil {
			s.logger.Info("receivedLeanHelixMessage() error decoding blockpair", log.Stringable("message-type", messageType), log.Error(err))
			return
		}
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
			s.logger.Info("receivedLeanHelixMessage() HandleLeanHelixMessage failed", log.Error(err))
		}
	}
}

func (s *service) SendLeanHelixMessage(ctx context.Context, input *gossiptopics.LeanHelixInput) (*gossiptopics.EmptyOutput, error) {

	messageType := input.Message.MessageType
	header := (&gossipmessages.HeaderBuilder{
		Topic:         gossipmessages.HEADER_TOPIC_LEAN_HELIX,
		LeanHelix:     messageType,
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
	}).Build()

	var blockPairPayloads [][]byte
	if input.Message.BlockPair != nil {
		var err error
		blockPairPayloads, err = encodeBlockPair(input.Message.BlockPair)
		if err != nil {
			s.logger.Info("gossip.SendLeanHelixMessage() ERROR", log.Error(err))
			return nil, err
		}
	}
	if input.Message.Content == nil {
		return nil, errors.Errorf("cannot encode LeanHelixMessage: %s", input.Message.String())
	}
	payloads := [][]byte{header.Raw(), input.Message.Content}
	if len(blockPairPayloads) > 0 {
		payloads = append(payloads, blockPairPayloads...)
	}
	return nil, s.transport.Send(ctx, &adapter.TransportData{
		SenderPublicKey: s.config.NodePublicKey(),
		RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_BROADCAST, // TODO: shouldn't be broadcast
		Payloads:        payloads,
	})
}

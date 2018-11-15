package gossip

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/gossip/codec"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

func (s *service) RegisterLeanHelixHandler(handler gossiptopics.LeanHelixHandler) {
	s.leanHelixHandlers = append(s.leanHelixHandlers, handler)
}

func (s *service) receivedLeanHelixMessage(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
	message, err := codec.DecodeLeanHelixMessage(header, payloads)
	if err != nil {
		return
	}

	for _, l := range s.leanHelixHandlers {
		_, err := l.HandleLeanHelixMessage(ctx, &gossiptopics.LeanHelixInput{Message: message})
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
	payloads, err := codec.EncodeLeanHelixMessage(header, input.Message)
	if err != nil {
		return nil, err
	}

	return nil, s.transport.Send(ctx, &adapter.TransportData{
		SenderPublicKey: s.config.NodePublicKey(),
		RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_BROADCAST, // TODO: shouldn't be broadcast
		Payloads:        payloads,
	})
}

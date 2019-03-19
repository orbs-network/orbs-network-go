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
	s.handlers.Lock()
	defer s.handlers.Unlock()

	s.handlers.leanHelixHandlers = append(s.handlers.leanHelixHandlers, handler)
}

func (s *service) receivedLeanHelixMessage(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
	message, err := codec.DecodeLeanHelixMessage(header, payloads)
	if err != nil {
		return
	}

	s.handlers.RLock()
	defer s.handlers.RUnlock()

	for _, l := range s.handlers.leanHelixHandlers {
		_, err := l.HandleLeanHelixMessage(ctx, &gossiptopics.LeanHelixInput{Message: message})
		if err != nil {
			s.logger.Info("receivedLeanHelixMessage() HandleLeanHelixMessage failed", log.Error(err))
		}
	}
}

func (s *service) SendLeanHelixMessage(ctx context.Context, input *gossiptopics.LeanHelixInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:          gossipmessages.HEADER_TOPIC_LEAN_HELIX,
		RecipientMode:  gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		VirtualChainId: s.config.VirtualChainId(),
	}).Build()

	payloads, err := codec.EncodeLeanHelixMessage(header, input.Message)
	if err != nil {
		return nil, err
	}

	return nil, s.transport.Send(ctx, &adapter.TransportData{
		SenderNodeAddress: s.config.NodeAddress(),
		RecipientMode:     gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		Payloads:          payloads,
	})
}

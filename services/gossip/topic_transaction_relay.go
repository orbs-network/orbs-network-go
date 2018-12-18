package gossip

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/gossip/codec"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

func (s *service) RegisterTransactionRelayHandler(handler gossiptopics.TransactionRelayHandler) {
	s.transactionHandlers = append(s.transactionHandlers, handler)
}

func (s *service) receivedTransactionRelayMessage(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
	switch header.TransactionRelay() {
	case gossipmessages.TRANSACTION_RELAY_FORWARDED_TRANSACTIONS:
		s.receivedForwardedTransactions(ctx, header, payloads)
	}
}

func (s *service) BroadcastForwardedTransactions(ctx context.Context, input *gossiptopics.ForwardedTransactionsInput) (*gossiptopics.EmptyOutput, error) {
	s.logger.Info("broadcasting forwarded transactions", trace.LogFieldFrom(ctx), log.Stringable("sender", input.Message.Sender), log.StringableSlice("transactions", input.Message.SignedTransactions))

	header := (&gossipmessages.HeaderBuilder{
		Topic:            gossipmessages.HEADER_TOPIC_TRANSACTION_RELAY,
		TransactionRelay: gossipmessages.TRANSACTION_RELAY_FORWARDED_TRANSACTIONS,
		RecipientMode:    gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
	}).Build()

	payloads, err := codec.EncodeForwardedTransactions(header, input.Message)
	if err != nil {
		return nil, err
	}

	return nil, s.transport.Send(ctx, &adapter.TransportData{
		SenderNodeAddress: s.config.NodeAddress(),
		RecipientMode:     gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		Payloads:          payloads,
	})
}

func (s *service) receivedForwardedTransactions(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	message, err := codec.DecodeForwardedTransactions(payloads)
	if err != nil {
		return
	}
	logger.Info("received forwarded transactions", log.Stringable("sender", message.Sender), log.StringableSlice("transactions", message.SignedTransactions))

	for _, l := range s.transactionHandlers {
		_, err := l.HandleForwardedTransactions(ctx, &gossiptopics.ForwardedTransactionsInput{Message: message})
		if err != nil {
			logger.Info("HandleForwardedTransactions failed", log.Error(err))
		}
	}
}

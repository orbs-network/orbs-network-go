package gossip

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
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

	payloads := make([][]byte, 0, 2+len(input.Message.SignedTransactions))
	payloads = append(payloads, header.Raw())
	payloads = append(payloads, input.Message.Sender.Raw())
	for _, tx := range input.Message.SignedTransactions {
		payloads = append(payloads, tx.Raw())
	}

	return nil, s.transport.Send(ctx, &adapter.TransportData{
		SenderPublicKey: s.config.NodePublicKey(),
		RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		Payloads:        payloads,
	})
}

func (s *service) receivedForwardedTransactions(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	txs := make([]*protocol.SignedTransaction, 0, len(payloads)-1)
	senderSignature := gossipmessages.SenderSignatureReader(payloads[0])

	for _, payload := range payloads[1:] {
		tx := protocol.SignedTransactionReader(payload)
		txs = append(txs, tx)
	}

	logger.Info("received forwarded transactions", log.Stringable("sender", senderSignature), log.StringableSlice("transactions", txs))

	for _, l := range s.transactionHandlers {
		_, err := l.HandleForwardedTransactions(ctx, &gossiptopics.ForwardedTransactionsInput{
			Message: &gossipmessages.ForwardedTransactionsMessage{
				Sender:             senderSignature,
				SignedTransactions: txs,
			},
		})
		if err != nil {
			logger.Info("HandleForwardedTransactions failed", log.Error(err))
		}
	}
}

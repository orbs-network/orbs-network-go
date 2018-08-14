package gossip

import (
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

func (s *service) RegisterTransactionRelayHandler(handler gossiptopics.TransactionRelayHandler) {
	s.transactionHandlers = append(s.transactionHandlers, handler)
}

func (s *service) receivedTransactionRelayMessage(header *gossipmessages.Header, payloads [][]byte) {
	switch header.TransactionRelay() {
	case gossipmessages.TRANSACTION_RELAY_FORWARDED_TRANSACTIONS:
		s.receivedForwardedTransactions(header, payloads)
	}
}

func (s *service) BroadcastForwardedTransactions(input *gossiptopics.ForwardedTransactionsInput) (*gossiptopics.EmptyOutput, error) {
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

	return nil, s.transport.Send(&adapter.TransportData{
		SenderPublicKey: s.config.NodePublicKey(),
		RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		Payloads:        payloads,
	})
}

func (s *service) receivedForwardedTransactions(header *gossipmessages.Header, payloads [][]byte) {
	txs := make([]*protocol.SignedTransaction, 0, len(payloads)-1)
	senderSignature := gossipmessages.SenderSignatureReader(payloads[0])

	for _, payload := range payloads[1:] {
		tx := protocol.SignedTransactionReader(payload)
		txs = append(txs, tx)
	}

	for _, l := range s.transactionHandlers {
		l.HandleForwardedTransactions(&gossiptopics.ForwardedTransactionsInput{
			Message: &gossipmessages.ForwardedTransactionsMessage{
				Sender:             senderSignature,
				SignedTransactions: txs,
			},
		})
	}
}

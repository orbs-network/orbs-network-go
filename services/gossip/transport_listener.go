package gossip

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"fmt"
)

func (s *service) OnTransportMessageReceived(message *gossipmessages.Header, payloads [][]byte) {
	s.reporting.Info(fmt.Sprintf("Gossip: OnMessageReceived [%s]", message))
	switch message.Topic() {
	case gossipmessages.HEADER_TOPIC_TRANSACTION_RELAY:
		s.receivedTransactionRelayMessage(message, payloads)
	case gossipmessages.HEADER_TOPIC_LEAN_HELIX:
		s.receivedLeanHelixMessage(message, payloads)
	}
}

func (s *service) receivedTransactionRelayMessage(message *gossipmessages.Header, payloads [][]byte) {
	switch message.TransactionRelay() {

	case gossipmessages.TRANSACTION_RELAY_FORWARDED_TRANSACTIONS:
		txs := make([]*protocol.SignedTransaction, 0, len(payloads))
		for _, payload := range payloads {
			tx := protocol.SignedTransactionReader(payload)
			txs = append(txs, tx)
		}
		for _, l := range s.transactionHandlers {
			l.HandleForwardedTransactions(&gossiptopics.ForwardedTransactionsInput{
				Message: &gossipmessages.ForwardedTransactionsMessage{
					SignedTransactions: txs,
				},
			})
		}

	}
}

func (s *service) receivedLeanHelixMessage(message *gossipmessages.Header, payloads [][]byte) {
	switch message.LeanHelix() {

	case gossipmessages.LEAN_HELIX_PRE_PREPARE:
		for _, l := range s.consensusHandlers {
			//l.OnVoteRequest(message.Sender, tx)
			l.HandleLeanHelixPrePrepare(&gossiptopics.LeanHelixPrePrepareInput{
				Message: &gossipmessages.LeanHelixPrePrepareMessage{
					BlockPair: &protocol.BlockPairContainer{
						TransactionsBlock: &protocol.TransactionsBlockContainer{
							SignedTransactions: []*protocol.SignedTransaction{protocol.SignedTransactionReader(payloads[0])},
						},
					},
				},
			})
		}

	case gossipmessages.LEAN_HELIX_PREPARE:
		for _, l := range s.consensusHandlers {
			l.HandleLeanHelixPrepare(&gossiptopics.LeanHelixPrepareInput{})
		}

	case gossipmessages.LEAN_HELIX_COMMIT:
		for _, l := range s.consensusHandlers {
			l.HandleLeanHelixCommit(&gossiptopics.LeanHelixCommitInput{})
		}

	}
}

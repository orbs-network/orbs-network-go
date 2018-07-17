package gossip

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

func (s *service) OnTransportMessageReceived(payloads [][]byte) {
	if len(payloads) == 0 {
		s.reporting.Error(&adapter.ErrCorruptData{})
		return
	}
	header := gossipmessages.HeaderReader(payloads[0])
	if !header.IsValid() {
		s.reporting.Error(&ErrCorruptHeader{payloads[0]})
		return
	}
	s.reporting.Info(fmt.Sprintf("Gossip: OnTransportMessageReceived: %s", header))
	switch header.Topic() {
	case gossipmessages.HEADER_TOPIC_TRANSACTION_RELAY:
		s.receivedTransactionRelayMessage(header, payloads[1:])
	case gossipmessages.HEADER_TOPIC_LEAN_HELIX:
		s.receivedLeanHelixMessage(header, payloads[1:])
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

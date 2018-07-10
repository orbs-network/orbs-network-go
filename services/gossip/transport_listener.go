package gossip

import (
	"fmt"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
)

func (s *service) OnTransportMessageReceived(message *protocol.GossipMessageHeader, payloads [][]byte) {
	fmt.Println("Gossip: OnMessageReceived", message)
	switch message.Topic() {
	case protocol.GossipMessageHeaderTopicTransactionRelayType:
		s.receivedTransactionRelayMessage(message, payloads)
	case protocol.GossipMessageHeaderTopicLeanHelixConsensusType:
		s.receivedLeanHelixConsensus(message, payloads)
	}
}

func (s *service) receivedTransactionRelayMessage(message *protocol.GossipMessageHeader, payloads [][]byte) {
	switch message.TransactionRelayType() {

	case gossipmessages.TRANSACTION_RELAY_FORWARDED_TRANSACTIONS:
		txs := make([]*protocol.SignedTransaction, 0, len(payloads))
		for _, payload := range payloads {
			tx := protocol.SignedTransactionReader(payload)
			txs = append(txs, tx)
		}
		for _, l := range s.transactionHandlers {
			l.HandleForwardedTransactions(&gossiptopics.ForwardedTransactionsInput{
				Transactions: txs,
			})
		}

	}
}

func (s *service) receivedLeanHelixConsensus(message *protocol.GossipMessageHeader, payloads [][]byte) {
	switch message.LeanHelixConsensusType() {

	case gossipmessages.LEAN_HELIX_CONSENSUS_PRE_PREPARE:
		for _, l := range s.consensusHandlers {
			//l.OnVoteRequest(message.Sender, tx)
			l.HandleLeanHelixPrePrepare(&gossiptopics.LeanHelixPrePrepareInput{
				Block:  payloads[0],
			})
		}

	case gossipmessages.LEAN_HELIX_CONSENSUS_PREPARE:
		for _, l := range s.consensusHandlers {
			l.HandleLeanHelixPrepare(&gossiptopics.LeanHelixPrepareInput{})
		}

	case gossipmessages.LEAN_HELIX_CONSENSUS_COMMIT:
		for _, l := range s.consensusHandlers {
			l.HandleLeanHelixCommit(&gossiptopics.LeanHelixCommitInput{})
		}

	}
}
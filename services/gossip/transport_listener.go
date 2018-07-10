package gossip

import (
	"fmt"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
)

func (s *service) OnTransportMessageReceived(message *adapter.Message) {
	fmt.Println("Gossip: OnMessageReceived", message)
	fmt.Println("Gossip: Message.Payload", message.Payload)

	switch message.Type {
	case adapter.CommitMessage:
		for _, l := range s.consensusHandlers {
			l.HandleLeanHelixCommit(&gossiptopics.LeanHelixCommitInput{})
		}

	case adapter.ForwardTransactionMessage:
		//TODO validate
		tx := protocol.SignedTransactionReader(message.Payload)
		if !tx.IsValid() {
			panic("invalid transaction!")
		}

		for _, l := range s.transactionHandlers {
			l.HandleForwardedTransactions(&gossiptopics.ForwardedTransactionsInput{Transactions: []*protocol.SignedTransaction{tx}})
		}

	case adapter.PrePrepareMessage:
		for _, l := range s.consensusHandlers {
			//l.OnVoteRequest(message.Sender, tx)
			prePrepareMessage := &gossiptopics.LeanHelixPrePrepareInput{
				Block:  message.Payload,
				Header: (&gossipmessages.LeanHelixPrePrepareHeaderBuilder{SenderPublicKey: []byte(message.Sender)}).Build(),
			}
			l.HandleLeanHelixPrePrepare(prePrepareMessage)
		}

	case adapter.PrepareMessage:
		for _, l := range s.consensusHandlers {
			l.HandleLeanHelixPrepare(&gossiptopics.LeanHelixPrepareInput{})
		}
	}
}
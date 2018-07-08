package gossip

import (
	"encoding/json"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type Config interface {
	NodeId() string
}

type Gossip interface {
	ForwardTransaction(transaction *protocol.SignedTransaction)
	CommitTransaction(transaction *protocol.SignedTransaction)
	RequestConsensusFor(transaction *protocol.SignedTransaction) error
	SendVote(candidate string, yay bool)

	RegisterTransactionListener(listener TransactionListener)
	RegisterConsensusListener(listener ConsensusListener)
}

type gossip struct {
	transport Transport

	transactionListeners     []TransactionListener
	consensusListeners       []ConsensusListener

	config Config
}

type TransactionListener interface {
	OnForwardTransaction(transaction *protocol.SignedTransaction)
}

type ConsensusListener interface {
	OnCommitTransaction(transaction *protocol.SignedTransaction)
	OnVote(voter string, yay bool)
	OnVoteRequest(originator string, transaction *protocol.SignedTransaction)
}

func NewGossip(transport Transport, config Config) Gossip {
	g := &gossip{transport: transport, config: config}
	transport.RegisterListener(g, g.config.NodeId())
	return g
}

func (g *gossip) RegisterTransactionListener(listener TransactionListener) {
	g.transactionListeners = append(g.transactionListeners, listener)
}

func (g *gossip) RegisterConsensusListener(listener ConsensusListener) {
	g.consensusListeners = append(g.consensusListeners, listener)
}

func (g *gossip) CommitTransaction(transaction *protocol.SignedTransaction) {
	g.transport.Broadcast(&Message{sender: g.config.NodeId(), Type: CommitMessage, payload: transaction.Raw()})
}

func (g *gossip) ForwardTransaction(transaction *protocol.SignedTransaction) {
	g.transport.Broadcast(&Message{sender: g.config.NodeId(), Type: ForwardTransactionMessage, payload: transaction.Raw()})
}

func (g *gossip) RequestConsensusFor(transaction *protocol.SignedTransaction) error {
	return g.transport.Broadcast(&Message{sender: g.config.NodeId(), Type: PrePrepareMessage, payload: transaction.Raw()})
}

func (g *gossip) SendVote(candidate string, yay bool) {
	bytes, _ := json.Marshal(yay)

	g.transport.Broadcast(&Message{sender: g.config.NodeId(), Type: PrepareMessage, payload: bytes})
}

func (g *gossip) OnMessageReceived(message *Message) {
	switch message.Type {
	case CommitMessage:
		//TODO validate
		tx := protocol.SignedTransactionReader(message.payload)
		if !tx.IsValid() {
			panic("invalid transaction!")
		}

		for _, l := range g.consensusListeners {
			l.OnCommitTransaction(tx)
		}

	case ForwardTransactionMessage:
		//TODO validate
		tx := protocol.SignedTransactionReader(message.payload)
		if !tx.IsValid() {
			panic("invalid transaction!")
		}

		for _, l := range g.transactionListeners {
			l.OnForwardTransaction(tx)
		}

	case PrePrepareMessage:
		//TODO validate
		tx := protocol.SignedTransactionReader(message.payload)
		if !tx.IsValid() {
			panic("invalid transaction!")
		}

		for _, l := range g.consensusListeners {
			l.OnVoteRequest(message.sender, tx)
		}

	case PrepareMessage:
		yay := false
		json.Unmarshal(message.payload, &yay)

		for _, l := range g.consensusListeners {
			l.OnVote(message.sender, yay)
		}
	}
}

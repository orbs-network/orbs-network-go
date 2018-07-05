package gossip

import (
	"github.com/orbs-network/orbs-network-go/types"
	"encoding/json"
)

type Config interface {
	NodeId() string
}

type Gossip interface {
	ForwardTransaction(transaction *types.Transaction)
	CommitTransaction(transaction *types.Transaction)
	RequestConsensusFor(transaction *types.Transaction) error
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
	OnForwardTransaction(transaction *types.Transaction)
}

type ConsensusListener interface {
	OnCommitTransaction(transaction *types.Transaction)
	OnVote(voter string, yay bool)
	OnVoteRequest(originator string, transaction *types.Transaction)
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

func (g *gossip) CommitTransaction(transaction *types.Transaction) {
	g.transport.Broadcast(Message{sender: g.config.NodeId(), Type: CommitMessage, payload: g.serialize(transaction)})
}

func (g *gossip) ForwardTransaction(transaction *types.Transaction) {
	g.transport.Broadcast(Message{sender: g.config.NodeId(), Type: ForwardTransactionMessage, payload: g.serialize(transaction)})
}

func (g *gossip) RequestConsensusFor(transaction *types.Transaction) error {
	return g.transport.Broadcast(Message{sender: g.config.NodeId(), Type: PrePrepareMessage, payload: g.serialize(transaction)})
}

func (g *gossip) SendVote(candidate string, yay bool) {
	g.transport.Broadcast(Message{sender: g.config.NodeId(), Type: PrepareMessage, payload: g.serialize(yay)})
}

func (g *gossip) OnMessageReceived(message Message) {
	switch message.Type {
	case CommitMessage:
		tx := &types.Transaction{}
		json.Unmarshal(message.payload, tx)

		for _, l := range g.consensusListeners {
			l.OnCommitTransaction(tx)
		}

	case ForwardTransactionMessage:
		tx := &types.Transaction{}
		json.Unmarshal(message.payload, tx)

		for _, l := range g.transactionListeners {
			l.OnForwardTransaction(tx)
		}

	case PrePrepareMessage:
		tx := &types.Transaction{}
		json.Unmarshal(message.payload, tx)

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

func (g *gossip) serialize(value interface{}) []byte {
	bytes, _ := json.Marshal(value)
	return bytes
}

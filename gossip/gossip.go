package gossip

import (
	"github.com/orbs-network/orbs-network-go/types"
	"encoding/json"
)

type Gossip interface {
	ForwardTransaction(transaction *types.Transaction)
	CommitTransaction(transaction *types.Transaction)
	RequestConsensusFor(transaction *types.Transaction) error
	BroadcastVote(yay bool)

	RegisterTransactionListener(listener TransactionListener)
	RegisterConsensusListener(listener ConsensusListener)
}

type gossip struct {
	transport Transport

	transactionListeners     []TransactionListener
	consensusListeners       []ConsensusListener
}

type TransactionListener interface {
	OnForwardTransaction(transaction *types.Transaction)
}

type ConsensusListener interface {
	OnCommitTransaction(transaction *types.Transaction)
	OnVote(yay bool)
	OnVoteRequest(transaction *types.Transaction)
}


func NewGossip(transport Transport) Gossip {
	g := &gossip{transport: transport}
	transport.RegisterListener(g)
	return g
}

func (g *gossip) RegisterTransactionListener(listener TransactionListener) {
	g.transactionListeners = append(g.transactionListeners, listener)
}

func (g *gossip) RegisterConsensusListener(listener ConsensusListener) {
	g.consensusListeners = append(g.consensusListeners, listener)
}

func (g *gossip) CommitTransaction(transaction *types.Transaction) {
	g.transport.Broadcast(CommitMessage, g.serialize(transaction))
}

func (g *gossip) ForwardTransaction(transaction *types.Transaction) {
	g.transport.Broadcast(ForwardTransactionMessage, g.serialize(transaction))
}

func (g *gossip) RequestConsensusFor(transaction *types.Transaction) error {
	return g.transport.Broadcast(PrePrepareMessage, g.serialize(transaction))
}

func (g *gossip) BroadcastVote(yay bool) {
	g.transport.Broadcast(PrepareMessage, g.serialize(yay))
}

func (g *gossip) OnMessageReceived(messageType string, bytes []byte) {
	switch messageType {
	case CommitMessage:
		tx := &types.Transaction{}
		json.Unmarshal(bytes, tx)

		for _, l := range g.consensusListeners {
			l.OnCommitTransaction(tx)
		}

	case ForwardTransactionMessage:
		tx := &types.Transaction{}
		json.Unmarshal(bytes, tx)

		for _, l := range g.transactionListeners {
			l.OnForwardTransaction(tx)
		}

	case PrePrepareMessage:
		tx := &types.Transaction{}
		json.Unmarshal(bytes, tx)

		for _, l := range g.consensusListeners {
			l.OnVoteRequest(tx)
		}

	case PrepareMessage:
		yay := false
		json.Unmarshal(bytes, &yay)

		for _, l := range g.consensusListeners {
			l.OnVote(yay)
		}
	}
}

func (g *gossip) serialize(value interface{}) []byte {
	bytes, _ := json.Marshal(value)
	return bytes
}

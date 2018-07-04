package gossip

import (
	"github.com/orbs-network/orbs-network-go/types"
	"encoding/json"
)

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

	nodeId string
}

type TransactionListener interface {
	OnForwardTransaction(transaction *types.Transaction)
}

type ConsensusListener interface {
	OnCommitTransaction(transaction *types.Transaction)
	OnVote(voter string, yay bool)
	OnVoteRequest(originator string, transaction *types.Transaction)
}


func NewGossip(transport Transport, nodeId string) Gossip {
	g := &gossip{transport: transport, nodeId: nodeId}
	transport.RegisterListener(g, g.nodeId)
	return g
}

func (g *gossip) RegisterTransactionListener(listener TransactionListener) {
	g.transactionListeners = append(g.transactionListeners, listener)
}

func (g *gossip) RegisterConsensusListener(listener ConsensusListener) {
	g.consensusListeners = append(g.consensusListeners, listener)
}

func (g *gossip) CommitTransaction(transaction *types.Transaction) {
	g.transport.Broadcast(g.nodeId, CommitMessage, g.serialize(transaction))
}

func (g *gossip) ForwardTransaction(transaction *types.Transaction) {
	g.transport.Broadcast(g.nodeId, ForwardTransactionMessage, g.serialize(transaction))
}

func (g *gossip) RequestConsensusFor(transaction *types.Transaction) error {
	return g.transport.Broadcast(g.nodeId, PrePrepareMessage, g.serialize(transaction))
}

func (g *gossip) SendVote(candidate string, yay bool) {
	g.transport.Unicast(g.nodeId, candidate, PrepareMessage, g.serialize(yay))
}

func (g *gossip) OnMessageReceived(sender string, messageType string, bytes []byte) {
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
			l.OnVoteRequest(sender, tx)
		}

	case PrepareMessage:
		yay := false
		json.Unmarshal(bytes, &yay)

		for _, l := range g.consensusListeners {
			l.OnVote(sender, yay)
		}
	}
}

func (g *gossip) serialize(value interface{}) []byte {
	bytes, _ := json.Marshal(value)
	return bytes
}

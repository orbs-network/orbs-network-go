package gossip

import (
	"encoding/json"
	"fmt"

	"github.com/orbs-network/orbs-network-go/types"
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

	transactionListeners []TransactionListener
	consensusListeners   []ConsensusListener

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
	g.transport.Broadcast(&Message{Sender: g.config.NodeId(), Type: CommitMessage, Payload: g.serialize(transaction)})
}

func (g *gossip) ForwardTransaction(transaction *types.Transaction) {
	g.transport.Broadcast(&Message{Sender: g.config.NodeId(), Type: ForwardTransactionMessage, Payload: g.serialize(transaction)})
}

func (g *gossip) RequestConsensusFor(transaction *types.Transaction) error {
	return g.transport.Broadcast(&Message{Sender: g.config.NodeId(), Type: PrePrepareMessage, Payload: g.serialize(transaction)})
}

func (g *gossip) SendVote(candidate string, yay bool) {
	message := Message{Sender: g.config.NodeId(), Type: PrepareMessage, Payload: g.serialize(yay)}
	fmt.Println("Sending vote", message)

	g.transport.Broadcast(&message)
}

func (g *gossip) OnMessageReceived(message *Message) {
	fmt.Println("Gossip: OnMessageReceived", message)
	fmt.Println("Gossip: Message.payload", message.Payload)

	switch message.Type {
	case CommitMessage:
		tx := &types.Transaction{}
		json.Unmarshal(message.Payload, tx)

		for _, l := range g.consensusListeners {
			l.OnCommitTransaction(tx)
		}

	case ForwardTransactionMessage:
		tx := &types.Transaction{}
		json.Unmarshal(message.Payload, tx)

		for _, l := range g.transactionListeners {
			l.OnForwardTransaction(tx)
		}

	case PrePrepareMessage:
		tx := &types.Transaction{}
		json.Unmarshal(message.Payload, tx)

		for _, l := range g.consensusListeners {
			l.OnVoteRequest(message.Sender, tx)
		}

	case PrepareMessage:
		yay := false
		// FIXME: always votes yes
		json.Unmarshal(message.Payload, &yay)

		fmt.Println(message.Sender, "votes", yay)

		for _, l := range g.consensusListeners {
			l.OnVote(message.Sender, yay)
		}
	}
}

func (g *gossip) serialize(value interface{}) []byte {
	bytes, _ := json.Marshal(value)
	return bytes
}

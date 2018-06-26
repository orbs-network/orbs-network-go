package bootstrap

import (
	"github.com/orbs-network/orbs-network-go/gossip"
	"github.com/orbs-network/orbs-network-go/ledger"
	"github.com/orbs-network/orbs-network-go/types"
)

type Node interface {
	gossip.Listener
	SendTransaction(transaction *types.Transaction)
	CallMethod() int
}

type node struct {
	gossip gossip.Gossip
	ledger ledger.Ledger
}

func NewNode(gossip gossip.Gossip) Node {
	return &node{
		gossip: gossip,
		ledger: ledger.NewLedger(),
	}
}

func (n *node) SendTransaction(transaction *types.Transaction) {
	n.gossip.ForwardTransaction(transaction)
}

func (n *node) CallMethod() int {
	return n.ledger.GetState()
}

func (n *node) OnForwardedTransaction(transaction *types.Transaction) error {
	n.ledger.AddTransaction(transaction)
	return nil
}

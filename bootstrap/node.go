package bootstrap

import (
	"github.com/orbs-network/orbs-network-go/gossip"
	"github.com/orbs-network/orbs-network-go/ledger"
)

type Node interface {
	gossip.Listener
	SendTransaction(value int)
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

func (n *node) SendTransaction(value int) {
	n.gossip.ForwardTransaction(value)
}

func (n *node) CallMethod() int {
	return n.ledger.GetState()
}

func (n *node) OnForwardedTransaction(value int) error {
	n.ledger.AddTransaction(value)
	return nil
}

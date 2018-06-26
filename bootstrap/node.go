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
	isLeader bool
	gossip   gossip.Gossip
	ledger   ledger.Ledger
}

func NewNode(gossip gossip.Gossip, isLeader bool) Node {
	return &node{
		isLeader: isLeader,
		gossip:   gossip,
		ledger:   ledger.NewLedger(),
	}
}

func (n *node) SendTransaction(transaction *types.Transaction) {
	if n.isLeader {
		n.gossip.CommitTransaction(transaction)
	} else {
		n.gossip.ForwardTransaction(transaction)
	}
}

func (n *node) CallMethod() int {
	return n.ledger.GetState()
}

func (n *node) OnForwardTransaction(transaction *types.Transaction) {
	if n.isLeader {
		n.gossip.CommitTransaction(transaction)
	}
}

func (n *node) OnCommitTransaction(transaction *types.Transaction) {
	n.ledger.AddTransaction(transaction)
}

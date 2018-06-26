package bootstrap

import "github.com/orbs-network/orbs-network-go/gossip"

type Node interface {
	gossip.Listener
	SendTransaction(value int) (int, error)
	CallMethod() (int, error)
}

type node struct {
	value int
	gossip gossip.Gossip
}

func NewNode(gossip gossip.Gossip) Node {
	return &node{
		gossip: gossip,
	}
}

func (n *node) SendTransaction(value int) (int, error) {
	n.gossip.ForwardTransaction(value)
	return n.value, nil
}

func (n *node) CallMethod() (int, error) {
	return n.value, nil
}

func (n *node) OnForwardedTransaction(value int) error {
	n.value = value
	return nil
}

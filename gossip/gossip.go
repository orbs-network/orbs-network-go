package gossip

import "github.com/orbs-network/orbs-network-go/types"

type Gossip interface {
	RegisterAll(listeners []Listener)
	ForwardTransaction(transaction *types.Transaction)
}

type gossip struct {
	listeners []Listener
}

func NewGossip() Gossip {
	return &gossip{}
}

func (g *gossip) RegisterAll(listeners []Listener) {
	g.listeners = listeners
}

func (g *gossip) ForwardTransaction(transaction *types.Transaction) {
	for _, l := range g.listeners {
		l.OnForwardedTransaction(transaction)
	}
}
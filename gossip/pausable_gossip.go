package gossip

import "github.com/orbs-network/orbs-network-go/types"

type PausableGossip interface {
	Gossip
	PauseForwards()
	ResumeForwards()
}

type pausableGossip struct {
	listeners []Listener
	paused bool
	pendingTransactions []types.Transaction
}

func NewPausableGossip() PausableGossip {
	return &pausableGossip{}
}

func (g *pausableGossip) RegisterAll(listeners []Listener) {
	g.listeners = listeners
}

func (g *pausableGossip) CommitTransaction(transaction *types.Transaction) {
	for _, l := range g.listeners {
		l.OnCommitTransaction(transaction)
	}
}

func (g *pausableGossip) ForwardTransaction(transaction *types.Transaction) {
	if g.paused {
		g.pendingTransactions = append(g.pendingTransactions, *transaction)
	} else {
		g.forwardToAllListeners(transaction)
	}
}

func (g *pausableGossip) forwardToAllListeners(transaction *types.Transaction) {
	for _, l := range g.listeners {
		l.OnForwardTransaction(transaction)
	}
}

func (g *pausableGossip) PauseForwards() {
	g.paused = true
}

func (g *pausableGossip) ResumeForwards() {
	g.paused = false
	for _, pendingTransaction := range g.pendingTransactions {
		g.forwardToAllListeners(&pendingTransaction)
	}
	g.pendingTransactions = nil
}
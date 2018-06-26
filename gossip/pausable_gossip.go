package gossip

import (
	"github.com/orbs-network/orbs-network-go/types"
	"sync"
)

type PausableGossip interface {
	Gossip
	PauseForwards()
	ResumeForwards()
	PauseConsensus()
	ResumeConsensus()
}

type pausableGossip struct {
	listeners           []Listener
	pausedForwards      bool
	pendingTransactions []types.Transaction
	consensusLock       *sync.Mutex
}

func NewPausableGossip() PausableGossip {
	return &pausableGossip{consensusLock: &sync.Mutex{}}
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
	if g.pausedForwards {
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
	g.pausedForwards = true
}

func (g *pausableGossip) ResumeForwards() {
	g.pausedForwards = false
	for _, pendingTransaction := range g.pendingTransactions {
		g.forwardToAllListeners(&pendingTransaction)
	}
	g.pendingTransactions = nil
}

func (g *pausableGossip) PauseConsensus() {
	g.consensusLock.Lock()
}

func (g *pausableGossip) ResumeConsensus() {
	g.consensusLock.Unlock()
}

func (g *pausableGossip) HasConsensusFor(transaction *types.Transaction) bool {
	g.consensusLock.Lock()
	defer g.consensusLock.Unlock()

	for _, l := range g.listeners {
		if !l.ValidateConsensusFor(transaction) {
			return false
		}
	}
	return true
}

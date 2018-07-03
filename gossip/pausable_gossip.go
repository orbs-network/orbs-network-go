package gossip

import (
	"github.com/orbs-network/orbs-network-go/types"
)

type PausableGossip interface {
	Gossip
	PauseForwards()
	ResumeForwards()
	FailConsensusRequests()
	PassConsensusRequests()
}

type pausableGossip struct {
	transactionListeners     []TransactionListener
	consensusListeners       []ConsensusListener
	pausedForwards           bool
	pendingTransactions      []types.Transaction
	failNextConsensusRequest bool
}

func NewPausableGossip() PausableGossip {
	return &pausableGossip{}
}

func (g *pausableGossip) RegisterTransactionListener(listener TransactionListener) {
	g.transactionListeners = append(g.transactionListeners, listener)
}

func (g *pausableGossip) RegisterConsensusListener(listener ConsensusListener) {
	g.consensusListeners = append(g.consensusListeners, listener)
}

func (g *pausableGossip) CommitTransaction(transaction *types.Transaction) {
	for _, l := range g.consensusListeners {
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
	for _, l := range g.transactionListeners {
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

func (g *pausableGossip) FailConsensusRequests() {
	g.failNextConsensusRequest = true
}

func (g *pausableGossip) PassConsensusRequests() {
	g.failNextConsensusRequest = false
}

func (g *pausableGossip) HasConsensusFor(transaction *types.Transaction) (bool, error) {
	if g.failNextConsensusRequest {
		return true, &ErrGossipRequestFailed{}
	}

	for _, l := range g.consensusListeners {
		if !l.ValidateConsensusFor(transaction) {
			return false, nil
		}
	}
	return true, nil
}

func (g *pausableGossip) RequestConsensusFor(transaction *types.Transaction) error {
	if g.failNextConsensusRequest {
		return &ErrGossipRequestFailed{}
	} else {
		for _, l := range g.consensusListeners {
			go l.OnVoteRequest(transaction)
		}

		return nil
	}

}

func (g *pausableGossip) BroadcastVote(yay bool) {
	for _, l := range g.consensusListeners {
		go l.OnVote(yay)
	}
}

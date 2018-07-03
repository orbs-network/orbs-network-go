package gossip

import "github.com/orbs-network/orbs-network-go/gossip"

type PausableTransport interface {
	gossip.Transport

	PauseForwards()
	ResumeForwards()
	FailConsensusRequests()
	PassConsensusRequests()
}

type pausableTransport struct {
	gossip.DispatchingTransport

	pausedForwards           bool
	pendingTransactions      [][]byte
	failNextConsensusRequest bool
}

func NewPausableGossip() PausableTransport {
	return &pausableTransport{}
}

func (g *pausableTransport) PauseForwards() {
	g.pausedForwards = true
}

func (g *pausableTransport) ResumeForwards() {
	g.pausedForwards = false
	for _, pending := range g.pendingTransactions {
		g.Broadcast(gossip.ForwardTransactionMessage, pending)
	}
	g.pendingTransactions = nil
}

func (g *pausableTransport) FailConsensusRequests() {
	g.failNextConsensusRequest = true
}

func (g *pausableTransport) PassConsensusRequests() {
	g.failNextConsensusRequest = false
}

func (g *pausableTransport) Broadcast(messageType string, bytes []byte) error {
	//TODO generalize pause / fail mechanism per message type
	if messageType == gossip.ForwardTransactionMessage && g.pausedForwards {
		g.pendingTransactions = append(g.pendingTransactions, bytes)
	} else if messageType == gossip.PrePrepareMessage && g.failNextConsensusRequest {
		return &gossip.ErrGossipRequestFailed{}
	} else {
		go g.receive(messageType, bytes)
	}

	return nil
}

func (g *pausableTransport) receive(messageType string, bytes []byte) {
	for _, l := range g.Listeners {
		l.OnMessageReceived(messageType, bytes)
	}
}


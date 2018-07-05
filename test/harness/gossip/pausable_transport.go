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
	listeners map[string]gossip.MessageReceivedListener

	pausedForwards           bool
	pendingMessages          []gossip.Message
	failNextConsensusRequest bool
}

func NewPausableTransport() PausableTransport {
	return &pausableTransport{listeners: make(map[string]gossip.MessageReceivedListener)}
}

func (t *pausableTransport) RegisterListener(listener gossip.MessageReceivedListener, myNodeId string) {
	t.listeners[myNodeId] = listener
}

func (g *pausableTransport) PauseForwards() {
	g.pausedForwards = true
}

func (g *pausableTransport) ResumeForwards() {
	g.pausedForwards = false
	for _, pending := range g.pendingMessages {
		g.Broadcast(pending)
	}
	g.pendingMessages = nil
}

func (g *pausableTransport) FailConsensusRequests() {
	g.failNextConsensusRequest = true
}

func (g *pausableTransport) PassConsensusRequests() {
	g.failNextConsensusRequest = false
}

func (g *pausableTransport) Broadcast(message gossip.Message) error {
	//TODO generalize pause / fail mechanism per message type
	if message.Type == gossip.ForwardTransactionMessage && g.pausedForwards {
		g.pendingMessages = append(g.pendingMessages, message)
	} else if message.Type == gossip.PrePrepareMessage && g.failNextConsensusRequest {
		return &gossip.ErrGossipRequestFailed{}
	} else {
		go g.receive(message)
	}

	return nil
}

//TODO pause unicasts as well as broadcasts
func (g *pausableTransport) Unicast(recipientId string, message gossip.Message) error {
	g.listeners[recipientId].OnMessageReceived(message)

	return nil
}

func (g *pausableTransport) receive(message gossip.Message) {
	for _, l := range g.listeners {
		l.OnMessageReceived(message)
	}
}


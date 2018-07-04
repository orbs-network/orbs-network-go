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
	pendingMessages          []message
	failNextConsensusRequest bool
}

type message struct {
	senderId string
	messageType string
	payload []byte
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
		g.Broadcast(pending.senderId, pending.messageType, pending.payload)
	}
	g.pendingMessages = nil
}

func (g *pausableTransport) FailConsensusRequests() {
	g.failNextConsensusRequest = true
}

func (g *pausableTransport) PassConsensusRequests() {
	g.failNextConsensusRequest = false
}

func (g *pausableTransport) Broadcast(senderId string, messageType string, bytes []byte) error {
	//TODO generalize pause / fail mechanism per message type
	if messageType == gossip.ForwardTransactionMessage && g.pausedForwards {
		g.pendingMessages = append(g.pendingMessages, message {senderId, messageType, bytes})
	} else if messageType == gossip.PrePrepareMessage && g.failNextConsensusRequest {
		return &gossip.ErrGossipRequestFailed{}
	} else {
		go g.receive(senderId, messageType, bytes)
	}

	return nil
}

//TODO pause unicasts as well as broadcasts
func (g *pausableTransport) Unicast(senderId string, recipientId, messageType string, bytes []byte) error {
	g.listeners[recipientId].OnMessageReceived(senderId, messageType, bytes)

	return nil
}

func (g *pausableTransport) receive(sender string, messageType string, bytes []byte) {
	for _, l := range g.listeners {
		l.OnMessageReceived(sender, messageType, bytes)
	}
}


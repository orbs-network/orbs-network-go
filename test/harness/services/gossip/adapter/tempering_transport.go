package adapter

import (
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
)

type TemperingTransport interface {
	adapter.Transport
	Pause(messagesOfType string)
	Resume(messagesOfType string)
	Fail(messagesOfType string)
	Pass(messagesOfType string)
}

type temperingTransport struct {
	listeners map[string]adapter.TransportListener
	pausedMessages map[string][]*adapter.Message
	failMessages   map[string]struct{}
}

func NewTemperingTransport() TemperingTransport {
	return &temperingTransport{
		listeners:      make(map[string]adapter.TransportListener),
		pausedMessages: make(map[string][]*adapter.Message),
		failMessages:   make(map[string]struct{}),
	}
}

func (t *temperingTransport) RegisterListener(listener adapter.TransportListener, myNodeId string) {
	t.listeners[myNodeId] = listener
}

func (g *temperingTransport) Pause(messagesOfType string) {
	g.pausedMessages[messagesOfType] = nil
}

func (g *temperingTransport) Resume(messagesOfType string) {
	messages, ok := g.pausedMessages[messagesOfType]
	if ok {
		for _, message := range messages {
			g.Broadcast(message)
		}
		delete(g.pausedMessages, messagesOfType)
	}
}

func (g *temperingTransport) Fail(messagesOfType string) {
	g.failMessages[messagesOfType] = struct{}{}
}

func (g *temperingTransport) Pass(messagesOfType string) {
	delete(g.failMessages, messagesOfType)
}

func (g *temperingTransport) Broadcast(message *adapter.Message) error {
	if g.paused(message.Type) {
		g.pausedMessages[message.Type] = append(g.pausedMessages[message.Type], message)
	} else if g.fail(message.Type) {
		return &adapter.ErrGossipRequestFailed{Message: *message}
	} else {
		go g.receive(*message)
	}

	return nil
}

//TODO pause/resume unicasts as well as broadcasts
func (g *temperingTransport) Unicast(recipientId string, message *adapter.Message) error {
	go g.listeners[recipientId].OnTransportMessageReceived(message)

	return nil
}

func (g *temperingTransport) receive(message adapter.Message) {
	for _, l := range g.listeners {
		l.OnTransportMessageReceived(&message)
	}
}

func (g *temperingTransport) paused(messageType string) bool {
	_, ok := g.pausedMessages[messageType]
	return ok
}

func (g *temperingTransport) fail(messageType string) bool {
	_, ok := g.failMessages[messageType]
	return ok
}

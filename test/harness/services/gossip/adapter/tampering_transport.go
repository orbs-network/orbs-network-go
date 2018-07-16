package adapter

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"sync"
)

type TamperingTransport interface {
	adapter.Transport
	Pause(topic gossipmessages.HeaderTopic, messageType uint16)
	Resume(topic gossipmessages.HeaderTopic, messageType uint16)
	Fail(topic gossipmessages.HeaderTopic, messageType uint16)
	Pass(topic gossipmessages.HeaderTopic, messageType uint16)
}

type messageWithPayloads struct {
	message  *gossipmessages.Header
	payloads [][]byte
}

type tamperingTransport struct {
	transportListeners map[string]adapter.TransportListener
	pausedMessages     map[string][]messageWithPayloads // TODO: this all needs to be synchronized
	failMessages       map[string]bool                  // TODO: this all needs to be synchronized
	mutex              *sync.Mutex
}

func NewTamperingTransport() TamperingTransport {
	return &tamperingTransport{
		transportListeners: make(map[string]adapter.TransportListener),
		pausedMessages:     make(map[string][]messageWithPayloads),
		failMessages:       make(map[string]bool),
		mutex:              &sync.Mutex{},
	}
}

func (t *tamperingTransport) RegisterListener(listener adapter.TransportListener, myNodeId string) {
	t.transportListeners[myNodeId] = listener
}

func (t *tamperingTransport) Send(header *gossipmessages.Header, payloads [][]byte) error {
	msgTypeStr := gossipMessageHeaderToTypeString(header)
	if t.fail(msgTypeStr) {
		return &adapter.ErrGossipRequestFailed{header}
	}
	if t.paused(msgTypeStr) {
		t.mutex.Lock()
		t.pausedMessages[msgTypeStr] = append(t.pausedMessages[msgTypeStr], messageWithPayloads{header, payloads})
		t.mutex.Unlock()
		return nil
	}
	go t.receive(header, payloads)
	return nil
}

func topicMessageToTypeString(topic gossipmessages.HeaderTopic, messageType uint16) string {
	return fmt.Sprintf("%d.%d", uint16(topic), messageType)
}

func gossipMessageHeaderMessageType(message *gossipmessages.Header) uint16 {
	switch message.Topic() {
	case gossipmessages.HEADER_TOPIC_TRANSACTION_RELAY:
		return uint16(message.TransactionRelay())
	case gossipmessages.HEADER_TOPIC_BLOCK_SYNC:
		return uint16(message.BlockSync())
	case gossipmessages.HEADER_TOPIC_LEAN_HELIX:
		return uint16(message.LeanHelix())
	}
	return 0
}

func gossipMessageHeaderToTypeString(message *gossipmessages.Header) string {
	messageType := gossipMessageHeaderMessageType(message)
	return fmt.Sprintf("%d.%d", uint16(message.Topic()), messageType)
}

func (t *tamperingTransport) Pause(topic gossipmessages.HeaderTopic, messageType uint16) {
	msgTypeStr := topicMessageToTypeString(topic, messageType)
	t.mutex.Lock()
	t.pausedMessages[msgTypeStr] = nil
	t.mutex.Unlock()
}

func (t *tamperingTransport) Resume(topic gossipmessages.HeaderTopic, messageType uint16) {
	msgTypeStr := topicMessageToTypeString(topic, messageType)
	t.mutex.Lock()
	messages, found := t.pausedMessages[msgTypeStr]
	delete(t.pausedMessages, msgTypeStr)
	t.mutex.Unlock()
	if found {
		for _, message := range messages {
			t.Send(message.message, message.payloads)
		}
	}
}

func (t *tamperingTransport) Fail(topic gossipmessages.HeaderTopic, messageType uint16) {
	messagesOfType := topicMessageToTypeString(topic, messageType)
	t.failMessages[messagesOfType] = true
}

func (t *tamperingTransport) Pass(topic gossipmessages.HeaderTopic, messageType uint16) {
	messagesOfType := topicMessageToTypeString(topic, messageType)
	delete(t.failMessages, messagesOfType)
}

func (t *tamperingTransport) receive(message *gossipmessages.Header, payloads [][]byte) {
	switch message.RecipientMode() {
	case gossipmessages.RECIPIENT_LIST_MODE_BROADCAST:
		for _, l := range t.transportListeners {
			// TODO: this is broadcasting to self
			l.OnTransportMessageReceived(message, payloads)
		}
	case gossipmessages.RECIPIENT_LIST_MODE_LIST:
		for i := message.RecipientPublicKeysIterator(); i.HasNext(); {
			nodeId := string(i.NextRecipientPublicKeys())
			t.transportListeners[nodeId].OnTransportMessageReceived(message, payloads)
		}
	case gossipmessages.RECIPIENT_LIST_MODE_ALL_BUT_LIST:
		panic("Not implemented")
	}
}

func (t *tamperingTransport) paused(msgTypeStr string) bool {
	t.mutex.Lock()
	_, found := t.pausedMessages[msgTypeStr]
	t.mutex.Unlock()
	return found
}

func (t *tamperingTransport) fail(msgTypeStr string) bool {
	_, found := t.failMessages[msgTypeStr]
	return found
}

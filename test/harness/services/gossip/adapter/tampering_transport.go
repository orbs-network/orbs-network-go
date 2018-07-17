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

type tamperingTransport struct {
	transportListeners map[string]adapter.TransportListener
	pausedMessages     map[string][]*adapter.TransportData
	failMessages       map[string]bool
	mutex              *sync.Mutex
}

func NewTamperingTransport() TamperingTransport {
	return &tamperingTransport{
		transportListeners: make(map[string]adapter.TransportListener),
		pausedMessages:     make(map[string][]*adapter.TransportData),
		failMessages:       make(map[string]bool),
		mutex:              &sync.Mutex{},
	}
}

func (t *tamperingTransport) RegisterListener(listener adapter.TransportListener, myNodeId string) {
	t.transportListeners[myNodeId] = listener
}

func (t *tamperingTransport) Send(data *adapter.TransportData) error {
	// because tampering requires intimate knowledge of the message, we must parse it (although we're not supposed to know its format)
	if data == nil || len(data.Payloads) == 0 {
		go t.receive(data) // according to contract we must always transport
		return nil
	}
	header := gossipmessages.HeaderReader(data.Payloads[0])
	if !header.IsValid() {
		go t.receive(data) // according to contract we must always transport
		return nil
	}
	msgTypeStr := gossipMessageHeaderToTypeString(header)
	if t.fail(msgTypeStr) {
		return &adapter.ErrTransportFailed{data}
	}
	if t.paused(msgTypeStr) {
		t.mutex.Lock()
		t.pausedMessages[msgTypeStr] = append(t.pausedMessages[msgTypeStr], data)
		t.mutex.Unlock()
		return nil
	}
	go t.receive(data)
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
		for _, data := range messages {
			t.Send(data)
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

func (t *tamperingTransport) receive(data *adapter.TransportData) {
	switch data.RecipientMode {
	case gossipmessages.RECIPIENT_LIST_MODE_BROADCAST:
		for _, l := range t.transportListeners {
			// TODO: this is broadcasting to self
			l.OnTransportMessageReceived(data.Payloads)
		}
	case gossipmessages.RECIPIENT_LIST_MODE_LIST:
		for _, recipientPublicKey := range data.RecipientPublicKeys {
			nodeId := string(recipientPublicKey)
			t.transportListeners[nodeId].OnTransportMessageReceived(data.Payloads)
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

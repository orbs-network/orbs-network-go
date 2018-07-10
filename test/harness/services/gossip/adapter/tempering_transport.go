package adapter

import (
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"fmt"
	"sync"
)

type TemperingTransport interface {
	adapter.Transport
	Pause(topic protocol.GossipMessageHeaderTopic, messageType uint16)
	Resume(topic protocol.GossipMessageHeaderTopic, messageType uint16)
	Fail(topic protocol.GossipMessageHeaderTopic, messageType uint16)
	Pass(topic protocol.GossipMessageHeaderTopic, messageType uint16)
}

type messageWithPayloads struct {
	message  *protocol.GossipMessageHeader
	payloads [][]byte
}

type temperingTransport struct {
	transportListeners map[string]adapter.TransportListener
	pausedMessages     map[string][]messageWithPayloads // TODO: this all needs to be synchronized
	failMessages       map[string]bool                  // TODO: this all needs to be synchronized
	mutex              *sync.Mutex
}

func NewTemperingTransport() TemperingTransport {
	return &temperingTransport{
		transportListeners: make(map[string]adapter.TransportListener),
		pausedMessages:     make(map[string][]messageWithPayloads),
		failMessages:       make(map[string]bool),
		mutex:              &sync.Mutex{},
	}
}

func (t *temperingTransport) RegisterListener(listener adapter.TransportListener, myNodeId string) {
	t.transportListeners[myNodeId] = listener
}

func (t *temperingTransport) Send(message *protocol.GossipMessageHeader, payloads [][]byte) error {
	msgTypeStr := gossipMessageHeaderToTypeString(message)
	if t.fail(msgTypeStr) {
		return &adapter.ErrGossipRequestFailed{message}
	}
	if t.paused(msgTypeStr) {
		t.mutex.Lock()
		t.pausedMessages[msgTypeStr] = append(t.pausedMessages[msgTypeStr], messageWithPayloads{message, payloads})
		t.mutex.Unlock()
		return nil
	}
	go t.receive(message, payloads)
	return nil
}

func topicMessageToTypeString(topic protocol.GossipMessageHeaderTopic, messageType uint16) string {
	return fmt.Sprintf("%d.%d", uint16(topic), messageType)
}

func gossipMessageHeaderMessageType(message *protocol.GossipMessageHeader) uint16 {
	switch (message.Topic()) {
	case protocol.GossipMessageHeaderTopicTransactionRelayType:
		return uint16(message.TransactionRelayType())
	case protocol.GossipMessageHeaderTopicBlockSyncType:
		return uint16(message.BlockSyncType())
	case protocol.GossipMessageHeaderTopicLeanHelixConsensusType:
		return uint16(message.LeanHelixConsensusType())
	}
	return 0
}

func gossipMessageHeaderToTypeString(message *protocol.GossipMessageHeader) string {
	messageType := gossipMessageHeaderMessageType(message)
	return fmt.Sprintf("%d.%d", uint16(message.Topic()), messageType)
}

func (t *temperingTransport) Pause(topic protocol.GossipMessageHeaderTopic, messageType uint16) {
	msgTypeStr := topicMessageToTypeString(topic, messageType)
	t.mutex.Lock()
	t.pausedMessages[msgTypeStr] = nil
	t.mutex.Unlock()
}

func (t *temperingTransport) Resume(topic protocol.GossipMessageHeaderTopic, messageType uint16) {
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

func (t *temperingTransport) Fail(topic protocol.GossipMessageHeaderTopic, messageType uint16) {
	messagesOfType := topicMessageToTypeString(topic, messageType)
	t.failMessages[messagesOfType] = true
}

func (t *temperingTransport) Pass(topic protocol.GossipMessageHeaderTopic, messageType uint16) {
	messagesOfType := topicMessageToTypeString(topic, messageType)
	delete(t.failMessages, messagesOfType)
}

func (t *temperingTransport) receive(message *protocol.GossipMessageHeader, payloads [][]byte) {
	switch message.RecipientMode() {
	case protocol.RECIPIENT_LIST_MODE_BROADCAST:
		for _, l := range t.transportListeners {
			// TODO: this is broadcasting to self
			l.OnTransportMessageReceived(message, payloads)
		}
	case protocol.RECIPIENT_LIST_MODE_SEND_TO_LIST:
		for i := message.RecipientPublicKeysIterator(); i.HasNext(); {
			nodeId := string(i.NextRecipientPublicKeys())
			t.transportListeners[nodeId].OnTransportMessageReceived(message, payloads)
		}
	case protocol.RECIPIENT_LIST_MODE_SEND_TO_ALL_BUT_LIST:
		panic("Not implemented")
	}
}

func (t *temperingTransport) paused(msgTypeStr string) bool {
	t.mutex.Lock()
	_, found := t.pausedMessages[msgTypeStr]
	t.mutex.Unlock()
	return found
}

func (t *temperingTransport) fail(msgTypeStr string) bool {
	_, found := t.failMessages[msgTypeStr]
	return found
}
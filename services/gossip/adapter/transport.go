package adapter

import (
	"fmt"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
)

type TransportListener interface {
	OnTransportMessageReceived(message *gossipmessages.Header, payloads [][]byte)
}

type Transport interface {
	RegisterListener(listener TransportListener, myNodeId string)
	Send(message *gossipmessages.Header, payloads [][]byte) error
}

type ErrGossipRequestFailed struct {
	Message *gossipmessages.Header
}

func (e *ErrGossipRequestFailed) Error() string {
	return fmt.Sprintf("gossip message failed to send: %v", e.Message)
}

package adapter

import (
	"fmt"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type TransportListener interface {
	OnTransportMessageReceived(message *protocol.GossipMessageHeader, payloads [][]byte)
}

type Transport interface {
	RegisterListener(listener TransportListener, myNodeId string)
	Send(message *protocol.GossipMessageHeader, payloads [][]byte) error
}

type ErrGossipRequestFailed struct {
	Message *protocol.GossipMessageHeader
}

func (e *ErrGossipRequestFailed) Error() string {
	return fmt.Sprintf("service message topic %v to %v has failed to send", e.Message.Topic(), e.Message.RecipientMode())
}

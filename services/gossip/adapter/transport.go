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
	return fmt.Sprintf("service message topic %v to %v has failed to send", e.Message.Topic(), e.Message.RecipientMode())
}

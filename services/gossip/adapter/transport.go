package adapter

import "fmt"

const CommitMessage = "Commit"
const ForwardTransactionMessage = "ForwardTx"
const PrePrepareMessage = "PrePrepare"
const PrepareMessage = "Prepare"

type Message struct {
	Sender  string
	Type    string // this is intentionally exported as pausable transport needs to be able to pause certain message types
	Payload []byte // use io.Writer / Reader?
}

type TransportListener interface {
	OnTransportMessageReceived(message *Message)
}

type Transport interface {
	Broadcast(message *Message) error
	Unicast(recipientId string, message *Message) error
	RegisterListener(listener TransportListener, myNodeId string)
}

type ErrGossipRequestFailed struct {
	Message Message
}

func (e *ErrGossipRequestFailed) Error() string {
	return fmt.Sprintf("service message [%s] to [%s] has failed to send", e.Message.Type, e.Message.Sender)
}

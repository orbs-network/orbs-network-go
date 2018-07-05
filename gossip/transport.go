package gossip

import "fmt"

const CommitMessage = "Commit"
const ForwardTransactionMessage = "ForwardTx"
const PrePrepareMessage = "PrePrepare"
const PrepareMessage = "Prepare"

type Message struct {
	sender  string
	Type    string // this is intentionally exported as pausable transport needs to be able to pause certain message types
	payload []byte
}

type MessageReceivedListener interface {
	OnMessageReceived(message Message)
}

type Transport interface {
	Broadcast(message Message) error
	Unicast(recipientId string, message Message) error

	RegisterListener(listener MessageReceivedListener, myNodeId string)
}

type ErrGossipRequestFailed struct {
	Message Message
}
func (e *ErrGossipRequestFailed) Error() string {
	return fmt.Sprintf("gossip message [%s] to [%s] has failed to send", e.Message.Type, e.Message.sender)
}


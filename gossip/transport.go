package gossip

const CommitMessage = "Commit"
const ForwardTransactionMessage = "ForwardTx"
const PrePrepareMessage = "PrePrepare"
const PrepareMessage = "Prepare"

type MessageReceivedListener interface {
	OnMessageReceived(sender string, messageType string, bytes []byte)
}

type Transport interface {
	Broadcast(senderId string, messageType string, payload []byte) error
	Unicast(senderId string, recipientId string, messageType string, payload []byte) error
	RegisterListener(listener MessageReceivedListener, myNodeId string)
}

type ErrGossipRequestFailed struct {}
func (e *ErrGossipRequestFailed) Error() string {
	return "the gossip request has failed"
}


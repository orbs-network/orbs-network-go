package gossip

const CommitMessage = "Commit"
const ForwardTransactionMessage = "ForwardTx"
const PrePrepareMessage = "PrePrepare"
const PrepareMessage = "Prepare"

type Message struct {
	sender  string
	Type    string
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

type ErrGossipRequestFailed struct {}
func (e *ErrGossipRequestFailed) Error() string {
	return "the gossip request has failed"
}


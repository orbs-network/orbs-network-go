package gossip

const CommitMessage = "Commit"
const ForwardTransactionMessage = "ForwardTx"
const PrePrepareMessage = "PrePrepare"
const PrepareMessage = "Prepare"

type MessageReceivedListener interface {
	OnMessageReceived(messageType string, bytes []byte)
}

type Transport interface {
	Broadcast(messageType string, bytes []byte) error
	RegisterListener(listener MessageReceivedListener)
}

type DispatchingTransport struct {
	Listeners []MessageReceivedListener
}

type ErrGossipRequestFailed struct {}
func (e *ErrGossipRequestFailed) Error() string {
	return "the gossip request has failed"
}

func (t *DispatchingTransport) RegisterListener(listener MessageReceivedListener) {
	t.Listeners = append(t.Listeners, listener)
}

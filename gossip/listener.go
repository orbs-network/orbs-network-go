package gossip

type Listener interface {
	OnForwardedTransaction(value int) error
}

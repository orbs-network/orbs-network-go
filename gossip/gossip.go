package gossip

import "github.com/orbs-network/orbs-network-go/types"

type Gossip interface {
	RegisterAll(listeners []Listener)
	ForwardTransaction(transaction *types.Transaction)
	CommitTransaction(transaction *types.Transaction)
	HasConsensusFor(transaction *types.Transaction) (bool, error)
}

type ErrGossipRequestFailed struct {}
func (e *ErrGossipRequestFailed) Error() string {
	return "the gossip request has failed"
}

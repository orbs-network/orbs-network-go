package gossip

import "github.com/orbs-network/orbs-network-go/types"

type Gossip interface {
	RegisterAll(listeners []Listener)
	ForwardTransaction(transaction *types.Transaction)
	CommitTransaction(transaction *types.Transaction)
	HasConsensusFor(transaction *types.Transaction) bool
}

package gossip

import "github.com/orbs-network/orbs-network-go/types"

type Listener interface {
	OnForwardTransaction(transaction *types.Transaction)
	OnCommitTransaction(transaction *types.Transaction)
}

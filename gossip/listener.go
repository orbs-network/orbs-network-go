package gossip

import "github.com/orbs-network/orbs-network-go/types"

type Listener interface {
	OnCommitTransaction(transaction *types.Transaction) error
}

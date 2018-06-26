package gossip

import "github.com/orbs-network/orbs-network-go/types"

type Listener interface {
	OnForwardedTransaction(transaction *types.Transaction) error
}

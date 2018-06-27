package gossip

import "github.com/orbs-network/orbs-network-go/types"

type TransactionListener interface {
	OnForwardTransaction(transaction *types.Transaction)
}

type ConsensusListener interface {
	OnCommitTransaction(transaction *types.Transaction)
	ValidateConsensusFor(transaction *types.Transaction) bool
}

package gossip

import "github.com/orbs-network/orbs-network-go/types"

type Gossip interface {
	ForwardTransaction(transaction *types.Transaction)
	CommitTransaction(transaction *types.Transaction)
	HasConsensusFor(transaction *types.Transaction) (bool, error)

	RegisterTransactionListener(listener TransactionListener)
	RegisterConsensusListener(listener ConsensusListener)
}

type ErrGossipRequestFailed struct {}
func (e *ErrGossipRequestFailed) Error() string {
	return "the gossip request has failed"
}

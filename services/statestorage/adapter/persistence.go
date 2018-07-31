package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type StatePersistence interface {
	WriteState(height primitives.BlockHeight, contract primitives.ContractName, stateDiffs *protocol.StateRecord) error // TODO: change this to an array as well since we do multiple writes in one transactions
	ReadState(height primitives.BlockHeight, contract primitives.ContractName) (map[string]*protocol.StateRecord, error)
}

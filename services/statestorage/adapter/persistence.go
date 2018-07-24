package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type StatePersistence interface {
	WriteState(contract primitives.ContractName, stateDiffs *protocol.StateRecord) // TODO: change this to an array as well since we do multiple writes in one transactions
	ReadState(contract primitives.ContractName) map[string]*protocol.StateRecord
}

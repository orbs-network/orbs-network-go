package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type StatePersistence interface {
	WriteState(height primitives.BlockHeight, contractStateDiffs []*protocol.ContractStateDiff) error
	ReadState(height primitives.BlockHeight, contract primitives.ContractName) (map[string]*protocol.StateRecord, error)
	Dump() string
}

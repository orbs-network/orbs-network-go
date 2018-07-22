package adapter

import (
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type InMemoryStatePersistence interface {
	adapter.StatePersistence
}

type inMemoryStatePersistence struct {
	stateWritten chan bool
	stateDiffs   map [primitives.ContractName][]*protocol.StateRecord
	config       adapter.Config
}

func NewInMemoryStatePersistence(config adapter.Config) adapter.StatePersistence {
	return &inMemoryStatePersistence{
		config:       config,
		stateDiffs:   make(map [primitives.ContractName][]*protocol.StateRecord),
		stateWritten: make(chan bool, 10),
	}
}

func (sp *inMemoryStatePersistence) WriteState(contract primitives.ContractName, stateDiff *protocol.StateRecord) {
	if contractStateDiff, ok := sp.stateDiffs[contract]; ok {
		sp.stateDiffs[contract] = append(contractStateDiff, stateDiff)
	} else {

		sp.stateDiffs[contract] = []*protocol.StateRecord{stateDiff}
	}
	sp.stateWritten <- true
}

func (sp *inMemoryStatePersistence) ReadState(contract primitives.ContractName, ) []*protocol.StateRecord {
	if contractStateDiff, ok := sp.stateDiffs[contract]; ok {
		return contractStateDiff
	} else {
		// TODO error ?
		return []*protocol.StateRecord{}
	}
}


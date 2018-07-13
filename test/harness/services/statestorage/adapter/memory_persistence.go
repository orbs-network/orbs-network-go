package adapter

import (
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type InMemoryStatePersistence interface {
	adapter.StatePersistence
}

type inMemoryStatePersistence struct {
	stateWritten chan bool
	stateDiffs   []*protocol.StateRecord
	config       adapter.Config
}

func NewInMemoryStatePersistence(config adapter.Config) adapter.StatePersistence {
	return &inMemoryStatePersistence{
		config:       config,
		stateWritten: make(chan bool, 10),
	}
}

func (sp *inMemoryStatePersistence) WriteState(stateDiff *protocol.StateRecord) {
	sp.stateDiffs = append(sp.stateDiffs, stateDiff)
	sp.stateWritten <- true
}

func (sp *inMemoryStatePersistence) ReadState() []*protocol.StateRecord {
	return sp.stateDiffs
}

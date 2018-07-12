package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
)

type Config interface {
	NodeId() string
}

type statePersistence struct {
	stateWritten chan bool
	stateDiffs        []protocol.StateRecord
	config       Config
}

func NewInMemoryBlockPersistence(config Config) adapter.StatePersistence {
	return &statePersistence{
		config:         config,
		stateWritten: make(chan bool, 10),
	}
}

func (sp *statePersistence) WriteState(stateDiff *protocol.StateRecord) {
	sp.stateDiffs = append(sp.stateDiffs, *stateDiff)
	sp.stateWritten <- true
}

func (sp *statePersistence) ReadState() []protocol.StateRecord {
	return sp.stateDiffs
}


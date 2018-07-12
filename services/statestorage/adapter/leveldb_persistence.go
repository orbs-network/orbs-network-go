package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type Config interface {
	NodeId() string
}

type statePersistence struct {
	stateWritten chan bool
	stateDiffs   []*protocol.StateRecord
	config       Config
}

func NewLevelDStatePersistence(config Config) StatePersistence {
	return &statePersistence{
		config:       config,
		stateWritten: make(chan bool, 10),
	}
}

func (sp *statePersistence) WriteState(stateDiff *protocol.StateRecord) {
	sp.stateDiffs = append(sp.stateDiffs, stateDiff)
	sp.stateWritten <- true
}

func (sp *statePersistence) ReadState() []*protocol.StateRecord {
	return sp.stateDiffs
}

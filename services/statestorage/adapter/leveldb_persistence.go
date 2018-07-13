package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type Config interface {
	NodeId() string
}

type levelDbStatePersistence struct {
	stateWritten chan bool
	stateDiffs   []*protocol.StateRecord
	config       Config
}

func NewLevelDbStatePersistence(config Config) StatePersistence {
	return &levelDbStatePersistence{
		config:       config,
		stateWritten: make(chan bool, 10),
	}
}

func (sp *levelDbStatePersistence) WriteState(stateDiff *protocol.StateRecord) {
	sp.stateDiffs = append(sp.stateDiffs, stateDiff)
	sp.stateWritten <- true
}

func (sp *levelDbStatePersistence) ReadState() []*protocol.StateRecord {
	return sp.stateDiffs
}

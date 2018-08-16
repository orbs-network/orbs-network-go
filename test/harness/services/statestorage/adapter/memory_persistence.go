package adapter

import (
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type InMemoryStatePersistence interface {
	adapter.StatePersistence
	Dump() string // TODO expose iterators/getters in StatePersistence and move Dump() to a wrapper struct in this file
	WaitUntilCommittedBlockOfHeight(height primitives.BlockHeight) error
}

func NewInMemoryStatePersistence() InMemoryStatePersistence {
	return adapter.NewInMemoryStatePersistence()
}

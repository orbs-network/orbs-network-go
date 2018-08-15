package adapter

import (
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
)

type InMemoryStatePersistence interface {
	adapter.StatePersistence
	Dump() string // TODO expose iterators/getters in StatePersistence and move Dump() to a wrapper struct in this file
}

func NewInMemoryStatePersistence() InMemoryStatePersistence {
	return adapter.NewInMemoryStatePersistence()
}

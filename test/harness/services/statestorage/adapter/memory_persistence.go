package adapter

import (
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
)

type InMemoryStatePersistence interface {
	adapter.StatePersistence
	Dump() string
}

func NewInMemoryStatePersistence() InMemoryStatePersistence {
	return adapter.NewInMemoryStatePersistence()
}

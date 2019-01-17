package testkit

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter/memory"
)

type DumpingStatePersistence interface {
	adapter.StatePersistence
	Dump() string
}

type TestStatePersistence struct {
	*memory.InMemoryStatePersistence
}

func NewDumpingStatePersistence(metric metric.Registry) *TestStatePersistence {
	result := &TestStatePersistence{
		InMemoryStatePersistence: memory.NewStatePersistence(metric),
	}
	return result
}

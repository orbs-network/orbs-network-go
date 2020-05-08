package serializer

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter/memory"
)

type StatePersistenceDeserializer interface {
	Deserialize([]byte) (adapter.StatePersistence, error)
}

type statePersistenceDeserializer struct {
	*memory.InMemoryStatePersistence
}

func NewPersistenceDeserializer(registry metric.Registry) StatePersistenceDeserializer {
	return &statePersistenceDeserializer{
		memory.NewStatePersistence(registry),
	}
}

func (s *statePersistenceDeserializer) Deserialize(raw []byte) (adapter.StatePersistence, error) {
	reader := SerializedMemoryPersistenceReader(raw)
	if !reader.IsValid() {
		return nil, fmt.Errorf("impossibe to deserialize state: invalid input")
	}

	blockHeight := reader.BlockHeight()
	timestamp := reader.Timestamp()
	refTime := reader.ReferenceTime()
	prevRefTime := reader.PreviousReferenceTime()
	proposer := reader.Proposer()
	merkle := reader.MerkleRootHash()

	for i := reader.EntriesIterator(); i.HasNext(); {
		entry := i.NextEntries()
		// FIXME could be optimized further
		diff := adapter.ChainState{entry.ContractName(): {string(entry.Key()): entry.Value()}}
		s.Write(blockHeight, timestamp, refTime, prevRefTime, proposer, merkle, diff)
	}

	return s.InMemoryStatePersistence, nil
}

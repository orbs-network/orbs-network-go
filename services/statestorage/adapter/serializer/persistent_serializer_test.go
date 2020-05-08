package serializer

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter/memory"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStatePersistenceSerializer_Dump(t *testing.T) {
	d := newDriver()
	d.checkIntegrity(t)

	serializer := NewStatePersistenceSerializer(d.InMemoryStatePersistence)
	dump, err := serializer.Dump()
	require.NoError(t, err)

	m, err := NewPersistenceDeserializer(metric.NewRegistry()).Deserialize(dump)
	require.NoError(t, err)

	deserializedDriver := newDriverFromPersistence(m.(*memory.InMemoryStatePersistence))
	deserializedDriver.checkIntegrity(t)
}

type driver struct {
	*memory.InMemoryStatePersistence
}

func newDriver() *driver {
	return &driver{
		memory.NewStatePersistence(metric.NewRegistry()),
	}
}

func newDriverFromPersistence(persistence *memory.InMemoryStatePersistence) *driver {
	return &driver{
		persistence,
	}
}

func (d *driver) writeSingleValueBlock(h primitives.BlockHeight, c, k, v string) error {
	diff := adapter.ChainState{primitives.ContractName(c): {k: []byte(v)}}
	return d.InMemoryStatePersistence.Write(h, 0, 0, 0, []byte("proposer"), []byte("merkle"), diff)
}

func (d *driver) checkIntegrity(t *testing.T) {
	d.writeSingleValueBlock(1, "foo", "foo", "bar")

	record, ok, err := d.Read("foo", "foo")
	require.NoError(t, err, "unexpected error")
	require.EqualValues(t, true, ok, "after writing a key it should exist")
	require.EqualValues(t, "bar", record, "after writing a key/value it should be returned")

}

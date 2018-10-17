package adapter

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReadStateWithNonExistingBlockHeight(t *testing.T) {
	d := NewInMemoryStatePersistence()
	_, _, err := d.ReadState(1, "foo", "")
	require.EqualError(t, err, "block 1 does not exist in snapshot history", "did not fail with error")
}

func TestReadStateWithNonExistingContractName(t *testing.T) {
	d := NewInMemoryStatePersistence()
	_, _, err := d.ReadState(0, "foo", "")
	require.NoError(t, err, "unexpected error")
}

func TestWriteStateAddAndRemoveKeyFromPersistentStorage(t *testing.T) {
	d := NewInMemoryStatePersistence()

	d.WriteState(1, []*protocol.ContractStateDiff{builders.ContractStateDiff().WithContractName("foo").WithStringRecord("foo", "bar").Build()})

	record, ok, err := d.ReadState(1, "foo", "foo")
	require.NoError(t, err, "unexpected error")
	require.EqualValues(t, true, ok, "after writing a key it should exist")
	require.EqualValues(t, "foo", record.Key(), "after writing a key/value it should be returned")
	require.EqualValues(t, "bar", record.Value(), "after writing a key/value it should be returned")

	d.WriteState(1, []*protocol.ContractStateDiff{builders.ContractStateDiff().WithContractName("foo").WithStringRecord("foo", "").Build()})

	_, ok, err = d.ReadState(1, "foo", "foo")
	require.NoError(t, err, "unexpected error")
	require.EqualValues(t, false, ok, "writing zero value to state did not remove key")
}

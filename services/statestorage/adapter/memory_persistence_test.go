package adapter

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReadStateWithNonExistingBlockHeight(t *testing.T) {
	d := NewInMemoryStatePersistence()
	_, err := d.ReadState(1, "foo")
	require.EqualError(t, err, "block 1 does not exist in snapshot history", "did not fail with error")
}

func TestReadStateWithNonExistingContractName(t *testing.T) {
	d := NewInMemoryStatePersistence()
	_, err := d.ReadState(0, "foo")
	require.EqualError(t, err, "contract foo does not exist", "did not fail with error")
}

func TestWriteState(t *testing.T) {
	d := NewInMemoryStatePersistence()

	d.WriteState(1, []*protocol.ContractStateDiff{builders.ContractStateDiff().WithContractName("foo").WithStringRecord("foo", "bar").Build()})

	records, err := d.ReadState(1, "foo")
	require.NoError(t, err, "unexpected error")
	require.Len(t, records, 1, "after writing one key there should be 1 keys")

	d.WriteState(1, []*protocol.ContractStateDiff{builders.ContractStateDiff().WithContractName("foo").WithStringRecord("foo", "").Build()})

	records, err = d.ReadState(1, "foo")
	require.NoError(t, err, "unexpected error")
	require.Len(t, records, 0, "writing zero value to state did not remove key")
}

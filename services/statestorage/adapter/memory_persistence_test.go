package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReadStateWithNonExistingBlockHeight(t *testing.T) {
	d := NewInMemoryStatePersistence()
	_, _, err := d.ReadState(1, "foo", "")
	require.EqualError(t, err, "block height mismatch. requested height 1, found 0", "did not fail with error")
}

func TestReadStateWithNonExistingContractName(t *testing.T) {
	d := NewInMemoryStatePersistence()
	_, _, err := d.ReadState(0, "foo", "")
	require.NoError(t, err, "unexpected error")
}

func TestWriteStateAddAndRemoveKeyFromPersistentStorage(t *testing.T) {
	d := NewInMemoryStatePersistence()

	d.WriteState(1, 0, []byte{}, buildDelta("foo","foo", "bar"))

	record, ok, err := d.ReadState(1, "foo", "foo")
	require.NoError(t, err, "unexpected error")
	require.EqualValues(t, true, ok, "after writing a key it should exist")
	require.EqualValues(t, "foo", record.Key(), "after writing a key/value it should be returned")
	require.EqualValues(t, "bar", record.Value(), "after writing a key/value it should be returned")

	d.WriteState(1, 0, []byte{}, buildDelta("foo","foo", ""))

	_, ok, err = d.ReadState(1, "foo", "foo")
	require.NoError(t, err, "unexpected error")
	require.EqualValues(t, false, ok, "writing zero value to state did not remove key")
}

func TestKeepLatestBlockOnly(t *testing.T) {
	d := NewInMemoryStatePersistence()

	d.WriteState(1, 0, []byte{}, buildDelta("foo","foo", "bar"))
	d.WriteState(2, 0, []byte{}, buildDelta("foo","baz", "qux"))

	_, _, err := d.ReadState(1, "foo", "foo")
	require.Error(t, err, "reading from outdated block height expected to fail")
	_, _, err = d.ReadState(3, "foo", "foo")
	require.Error(t, err, "reading from future block height expected to fail")
}


func buildDelta(contract, key, value string) (map[string]map[string]*protocol.StateRecord){
	record := (&protocol.StateRecordBuilder{Key: []byte(key), Value: []byte(value)}).Build()
	return map[string]map[string]*protocol.StateRecord{contract: {key: record}}
}
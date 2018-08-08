package virtualmachine

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
)

func requireDirtyPairs(t *testing.T, s *transientState, contract primitives.ContractName, expected []keyValuePair) {
	d := []keyValuePair{}
	s.forDirty("Contract1", func(key []byte, value []byte) {
		d = append(d, keyValuePair{key, value, true})
	})
	require.ElementsMatch(t, expected, d, "dirty keys should be equal")
}

func TestTransientStateReadMissingContract(t *testing.T) {
	s := newTransientState()

	_, found := s.getValue("Contract1", []byte{0x01})
	require.False(t, found, "key should not be found")

	requireDirtyPairs(t, s, "Contract1", []keyValuePair{})
}

func TestTransientStateReadMissingKey(t *testing.T) {
	s := newTransientState()
	s.setValue("Contract1", []byte{0x02}, []byte{0x77, 0x88}, false)

	_, found := s.getValue("Contract1", []byte{0x01})
	require.False(t, found, "key should not be found")

	requireDirtyPairs(t, s, "Contract1", []keyValuePair{})
}

func TestTransientStateWriteReadKey(t *testing.T) {
	s := newTransientState()
	s.setValue("Contract1", []byte{0x01}, []byte{0x77, 0x88}, false)

	v, found := s.getValue("Contract1", []byte{0x01})
	require.True(t, found, "key should be found")
	require.Equal(t, []byte{0x77, 0x88}, v, "value should be equal")

	requireDirtyPairs(t, s, "Contract1", []keyValuePair{})
}

func TestTransientStateReplaceKey(t *testing.T) {
	s := newTransientState()
	s.setValue("Contract1", []byte{0x01}, []byte{0x77, 0x88}, false)
	s.setValue("Contract1", []byte{0x01}, []byte{0x99, 0xaa, 0xbb}, false)

	v, found := s.getValue("Contract1", []byte{0x01})
	require.True(t, found, "key should be found")
	require.Equal(t, []byte{0x99, 0xaa, 0xbb}, v, "value should be equal")

	requireDirtyPairs(t, s, "Contract1", []keyValuePair{})
}

func TestTransientStateWriteDirtyReadKeys(t *testing.T) {
	s := newTransientState()
	s.setValue("Contract1", []byte{0x01}, []byte{0x22, 0x33}, true)
	s.setValue("Contract1", []byte{0x02}, []byte{0x33, 0x44}, false)
	s.setValue("Contract1", []byte{0x03}, []byte{0x44, 0x55}, false)
	s.setValue("Contract1", []byte{0x03}, []byte{0x55, 0x66}, true)
	s.setValue("Contract1", []byte{0x04}, []byte{0x66, 0x77}, true)
	s.setValue("Contract1", []byte{0x04}, []byte{0x77, 0x88}, false)
	s.setValue("Contract1", []byte{0x05}, []byte{0x88, 0x99}, true)
	s.setValue("Contract1", []byte{0x05}, []byte{0x99, 0xaa}, true)

	v, found := s.getValue("Contract1", []byte{0x01})
	require.True(t, found, "key should be found")
	require.Equal(t, []byte{0x22, 0x33}, v, "value should be equal")

	requireDirtyPairs(t, s, "Contract1", []keyValuePair{
		{[]byte{0x01}, []byte{0x22, 0x33}, true},
		{[]byte{0x03}, []byte{0x55, 0x66}, true},
		{[]byte{0x05}, []byte{0x99, 0xaa}, true},
	})
}

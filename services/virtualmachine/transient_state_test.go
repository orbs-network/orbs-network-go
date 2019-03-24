// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package virtualmachine

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTransientState_ReadMissingContract(t *testing.T) {
	s := newTransientState()

	_, found := s.getValue("Contract1", []byte{0x01})
	require.False(t, found, "key should not be found")

	requireDirtyPairs(t, s, "Contract1", []keyValuePair{})
}

func TestTransientState_ReadMissingKey(t *testing.T) {
	s := newTransientState()
	s.setValue("Contract1", []byte{0x02}, []byte{0x77, 0x88}, false)

	_, found := s.getValue("Contract1", []byte{0x01})
	require.False(t, found, "key should not be found")

	requireDirtyPairs(t, s, "Contract1", []keyValuePair{})
}

func TestTransientState_WriteReadKey(t *testing.T) {
	s := newTransientState()
	s.setValue("Contract1", []byte{0x01}, []byte{0x77, 0x88}, false)

	v, found := s.getValue("Contract1", []byte{0x01})
	require.True(t, found, "key should be found")
	require.Equal(t, []byte{0x77, 0x88}, v, "value should be equal")

	require.EqualValues(t, []primitives.ContractName{"Contract1"}, s.contractSortOrder, "contract sort order should match")
	requireDirtyPairs(t, s, "Contract1", []keyValuePair{})
}

func TestTransientState_ReplaceKey(t *testing.T) {
	s := newTransientState()
	s.setValue("Contract1", []byte{0x01}, []byte{0x77, 0x88}, false)
	s.setValue("Contract1", []byte{0x01}, []byte{0x99, 0xaa, 0xbb}, false)

	v, found := s.getValue("Contract1", []byte{0x01})
	require.True(t, found, "key should be found")
	require.Equal(t, []byte{0x99, 0xaa, 0xbb}, v, "value should be equal")

	require.EqualValues(t, []primitives.ContractName{"Contract1"}, s.contractSortOrder, "contract sort order should match")
	requireDirtyPairs(t, s, "Contract1", []keyValuePair{})
}

func TestTransientState_WriteDirtyReadKeys(t *testing.T) {
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

	require.EqualValues(t, []primitives.ContractName{"Contract1"}, s.contractSortOrder, "contract sort order should match")
	requireDirtyPairs(t, s, "Contract1", []keyValuePair{
		{[]byte{0x01}, []byte{0x22, 0x33}, true},
		{[]byte{0x03}, []byte{0x55, 0x66}, true},
		{[]byte{0x05}, []byte{0x99, 0xaa}, true},
	})
}

func TestTransientState_Merge(t *testing.T) {
	s1 := newTransientState()
	s1.setValue("Contract1", []byte{0x01}, []byte{0x22, 0x33}, true)
	s1.setValue("Contract1", []byte{0x02}, []byte{0x44, 0x55}, true)
	s1.setValue("Contract4", []byte{0x04}, []byte{0xff, 0xff}, true)

	s2 := newTransientState()
	s2.setValue("Contract1", []byte{0x02}, []byte{0x66, 0x77, 0x88}, true)
	s2.setValue("Contract1", []byte{0x03}, []byte{0x99}, true)
	s2.setValue("Contract2", []byte{0x01}, []byte{0xaa}, true)
	s2.setValue("Contract3", []byte{0x01}, []byte{0x11}, true)

	s2.mergeIntoTransientState(s1)

	require.EqualValues(t, []primitives.ContractName{"Contract1", "Contract4", "Contract2", "Contract3"}, s1.contractSortOrder, "contract sort order should match")
	requireDirtyPairs(t, s1, "Contract1", []keyValuePair{
		{[]byte{0x01}, []byte{0x22, 0x33}, true},
		{[]byte{0x02}, []byte{0x66, 0x77, 0x88}, true},
		{[]byte{0x03}, []byte{0x99}, true},
	})
	requireDirtyPairs(t, s1, "Contract2", []keyValuePair{
		{[]byte{0x01}, []byte{0xaa}, true},
	})
	requireDirtyPairs(t, s1, "Contract3", []keyValuePair{
		{[]byte{0x01}, []byte{0x11}, true},
	})
	requireDirtyPairs(t, s1, "Contract4", []keyValuePair{
		{[]byte{0x04}, []byte{0xff, 0xff}, true},
	})
}

func TestTransientState_DirtyKeys_DeterministicSortOrder(t *testing.T) {
	s := newTransientState()
	s.setValue("Contract3", []byte{0x03}, []byte{}, true)
	s.setValue("Contract1", []byte{0x01}, []byte{}, false)
	s.setValue("Contract2", []byte{0x02}, []byte{}, true)
	s.setValue("Contract3", []byte{0x02}, []byte{}, true)
	s.setValue("Contract1", []byte{0x02}, []byte{}, false)
	s.setValue("Contract2", []byte{0x02}, []byte{0x11}, true)
	s.setValue("Contract3", []byte{0x01}, []byte{}, true)
	s.setValue("Contract1", []byte{0x03}, []byte{}, false)
	s.setValue("Contract2", []byte{0x02}, []byte{0x22}, true)

	require.EqualValues(t, []primitives.ContractName{"Contract3", "Contract1", "Contract2"}, s.contractSortOrder, "contract sort order should match")
	requireDirtyPairs(t, s, "Contract3", []keyValuePair{
		{[]byte{0x03}, []byte{}, true},
		{[]byte{0x02}, []byte{}, true},
		{[]byte{0x01}, []byte{}, true},
	})
	requireDirtyPairs(t, s, "Contract1", []keyValuePair{})
	requireDirtyPairs(t, s, "Contract2", []keyValuePair{
		{[]byte{0x02}, []byte{0x22}, true},
	})
}

func requireDirtyPairs(t *testing.T, s *transientState, contract primitives.ContractName, expected []keyValuePair) {
	d := []keyValuePair{}
	s.forDirty(contract, func(key []byte, value []byte) {
		d = append(d, keyValuePair{key, value, true})
	})
	require.EqualValues(t, expected, d, "dirty keys should be equal")
}

// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package memory

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReadStateWithNonExistingContractName(t *testing.T) {
	d := NewStatePersistence(metric.NewRegistry())
	_, _, err := d.Read("foo", "")
	require.NoError(t, err, "unexpected error")
}

func TestWriteStateAddAndRemoveKeyFromPersistentStorage(t *testing.T) {
	d := newDriver()

	d.writeSingleValueBlock(1, "foo", "foo", "bar")

	record, ok, err := d.Read("foo", "foo")
	require.NoError(t, err, "unexpected error")
	require.EqualValues(t, true, ok, "after writing a key it should exist")
	require.EqualValues(t, "foo", record.Key(), "after writing a key/value it should be returned")
	require.EqualValues(t, "bar", record.Value(), "after writing a key/value it should be returned")

	d.writeSingleValueBlock(1, "foo", "foo", "")

	_, ok, err = d.Read("foo", "foo")
	require.NoError(t, err, "unexpected error")
	require.EqualValues(t, false, ok, "writing zero value to state did not remove key")
}

type driver struct {
	*InMemoryStatePersistence
}

func newDriver() *driver {
	return &driver{
		NewStatePersistence(metric.NewRegistry()),
	}
}

func (d *driver) writeSingleValueBlock(h primitives.BlockHeight, c, k, v string) error {
	record := (&protocol.StateRecordBuilder{Key: []byte(k), Value: []byte(v)}).Build()
	diff := adapter.ChainState{primitives.ContractName(c): {k: record}}
	return d.InMemoryStatePersistence.Write(h, 0, []byte{}, diff)
}

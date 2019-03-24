// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package statestorage

import (
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/crypto/merkle"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestWriteAtHeight(t *testing.T) {
	persistenceMock := statePersistenceMockWithWriteAnyNoErrors(0)
	d := newDriver(t, persistenceMock, 5, nil)
	persistenceMock.
		When("Read", primitives.ContractName("c"), "k1").
		Return((*protocol.StateRecord)(nil), false, nil).
		Times(1)

	d.write(1, "c", "k1", "v1")

	_, exists, err := d.read(0, "c", "k1")
	require.NoError(t, err)
	require.EqualValues(t, false, exists)

	valueAtHeight1, exists, err := d.read(1, "c", "k1")
	require.NoError(t, err)
	require.EqualValues(t, true, exists)
	require.EqualValues(t, "v1", valueAtHeight1)

	_, _, err = d.read(200, "c", "k1")
	require.Error(t, err)
	require.EqualError(t, err, "requested height 200 is too new. most recent available block height is 1")

	_, errCalled := persistenceMock.Verify()
	require.NoError(t, errCalled, "error happened when it should not")

}

func TestNoLayers(t *testing.T) {
	persistenceMock := &StatePersistenceMock{}
	persistenceMock.
		When("Write", mock.Any, mock.Any, mock.Any, mock.Any).
		Return(nil).
		Times(2)
	d := newDriver(t, persistenceMock, 0, nil)
	d.writeFull(1, 1, "c", "k", "v1")
	d.writeFull(2, 2, "c", "k", "v2")

	_, _, err := d.read(1, "c", "k")
	require.EqualError(t, err, "requested height 1 is too old. oldest available block height is 2")

	_, errCalled := persistenceMock.Verify()
	require.NoError(t, errCalled, "error happened when it should not")

}

func TestWriteAtHeightAndDeleteAtLaterHeight(t *testing.T) {
	d := newDriver(t, statePersistenceMockWithWriteAnyNoErrors(0), 5, nil)
	d.write(1, "", "k1", "v1")
	d.write(2, "", "k1", "")

	valueAtHeight1, exists, err := d.read(1, "", "k1")
	require.NoError(t, err)
	require.EqualValues(t, true, exists)
	require.EqualValues(t, "v1", valueAtHeight1)

	valueAtHeight2, exists, err := d.read(2, "", "k1")
	require.NoError(t, err)
	require.EqualValues(t, false, exists)
	require.EqualValues(t, "", valueAtHeight2)
}

func TestMergeToPersistence(t *testing.T) {
	var writeCallCount byte = 1
	persistenceMock := &StatePersistenceMock{}
	persistenceMock.
		When("Write", mock.Any, mock.Any, mock.Any, mock.Any).
		Call(func(height primitives.BlockHeight, ts primitives.TimestampNano, root primitives.Sha256, diff adapter.ChainState) error {
			expectedValue := fmt.Sprintf("v%v", writeCallCount)
			v := string(diff["c"]["k"].Value())
			require.EqualValues(t, expectedValue, v)
			require.EqualValues(t, writeCallCount, height)
			require.EqualValues(t, writeCallCount, ts)
			require.EqualValues(t, primitives.Sha256{writeCallCount}, root)
			writeCallCount++
			return nil
		}).
		Times(2)
	d := newDriver(t, persistenceMock, 2, nil)
	d.writeFull(1, 1, "c", "k", "v1")
	d.writeFull(2, 2, "c", "k", "v2")
	d.writeFull(3, 3, "c", "k", "v3")
	d.writeFull(4, 4, "c", "k", "v4")

	_, errCalled := persistenceMock.Verify()
	require.NoError(t, errCalled, "error happened when it should not")
}

func TestReadOutOfRange(t *testing.T) {
	persistenceMock := statePersistenceMockWithWriteAnyNoErrors(2)
	d := newDriver(t, persistenceMock, 2, nil)
	d.writeFull(1, 1, "c", "k", "v1")
	d.writeFull(2, 2, "c", "k", "v2")
	d.writeFull(3, 3, "c", "k", "v3")
	d.writeFull(4, 4, "c", "k", "v4")

	_, _, err := d.read(1, "c", "k")
	require.EqualError(t, err, "requested height 1 is too old. oldest available block height is 2")

	_, err = d.readHash(1)
	require.EqualError(t, err, "could not locate merkle hash for height 1. oldest available block height is 2")

	_, errCalled := persistenceMock.Verify()
	require.NoError(t, errCalled, "error happened when it should not")
}

func TestReadHash(t *testing.T) {
	persistenceMock := statePersistenceMockWithWriteAnyNoErrors(1)
	d := newDriver(t, persistenceMock, 1, nil)
	d.writeFull(1, 1, "c", "k", "v1")
	d.writeFull(2, 2, "c", "k", "v2")

	root, err := d.readHash(1)
	require.NoError(t, err)
	require.Equal(t, primitives.Sha256{1}, root)

	root, err = d.readHash(2)
	require.NoError(t, err)
	require.Equal(t, primitives.Sha256{2}, root)

	_, err = d.readHash(3)
	require.Error(t, err)

	_, errCalled := persistenceMock.Verify()
	require.NoError(t, errCalled, "error happened when it should not")
}

func TestRevisionEviction(t *testing.T) {
	persistenceMock := statePersistenceMockWithWriteAnyNoErrors(1)
	var evictedMerkleRoots []primitives.Sha256
	d := newDriver(t, persistenceMock, 1, func(sha256 primitives.Sha256) {
		evictedMerkleRoots = append(evictedMerkleRoots, sha256)
	})

	firstHash, _ := d.readHash(0)
	d.writeFull(1, 1, "c", "k", "v1")
	require.Len(t, evictedMerkleRoots, 0)

	d.writeFull(2, 2, "c", "k", "v2")
	require.Equal(t, []primitives.Sha256{firstHash}, evictedMerkleRoots)
}

type driver struct {
	inner *rollingRevisions
}

func newDriver(tb testing.TB, persistence adapter.StatePersistence, layers int, merkleForgetCallback func(sha256 primitives.Sha256)) *driver {
	m := newMerkleMock()
	if merkleForgetCallback != nil {
		m.When("Forget", mock.Any).Call(merkleForgetCallback).Return(nil).Times(1)
	} else {
		m.When("Forget", mock.Any).Return(nil).Times(1)
	}
	d := &driver{
		inner: newRollingRevisions(log.DefaultTestingLogger(tb), persistence, layers, m),
	}
	return d
}

func (d *driver) write(h primitives.BlockHeight, contract primitives.ContractName, kv ...string) error {
	diff := adapter.ChainState{contract: make(adapter.ContractState)}
	for i := 0; i < len(kv); i += 2 {
		diff[contract][kv[i]] = (&protocol.StateRecordBuilder{Key: []byte(kv[i]), Value: []byte(kv[i+1])}).Build()
	}
	return d.inner.addRevision(h, 0, diff)
}

func (d *driver) writeFull(h primitives.BlockHeight, ts primitives.TimestampNano, contract primitives.ContractName, kv ...string) error {
	diff := adapter.ChainState{contract: make(adapter.ContractState)}
	for i := 0; i < len(kv); i += 2 {
		diff[contract][kv[i]] = (&protocol.StateRecordBuilder{Key: []byte(kv[i]), Value: []byte(kv[i+1])}).Build()
	}
	return d.inner.addRevision(h, ts, diff)
}

func (d *driver) read(h primitives.BlockHeight, contract primitives.ContractName, key string) (string, bool, error) {
	r, exists, err := d.inner.getRevisionRecord(h, contract, key)
	value := ""
	if r != nil {
		value = string(r.Value())
	}
	return value, exists, err
}

func (d *driver) readHash(h primitives.BlockHeight) (primitives.Sha256, error) {
	return d.inner.getRevisionHash(h)
}

type StatePersistenceMock struct {
	mock.Mock
}

func statePersistenceMockWithWriteAnyNoErrors(writeTimes int) *StatePersistenceMock {
	persistenceMock := &StatePersistenceMock{}
	persistenceMock.
		When("Write", mock.Any, mock.Any, mock.Any, mock.Any).
		Return(nil).
		Times(writeTimes)
	return persistenceMock
}

func (spm *StatePersistenceMock) Write(height primitives.BlockHeight, ts primitives.TimestampNano, root primitives.Sha256, diff adapter.ChainState) error {
	return spm.Mock.Called(height, ts, root, diff).Error(0)
}
func (spm *StatePersistenceMock) Read(contract primitives.ContractName, key string) (*protocol.StateRecord, bool, error) {
	ret := spm.Mock.Called(contract, key)
	return ret.Get(0).(*protocol.StateRecord), ret.Bool(1), ret.Error(2)
}
func (spm *StatePersistenceMock) ReadMetadata() (primitives.BlockHeight, primitives.TimestampNano, primitives.Sha256, error) {
	return 0, 0, primitives.Sha256{}, nil
}

type MerkleMock struct {
	mock.Mock
}

func newMerkleMock() *MerkleMock {
	m := &MerkleMock{}
	var counter byte = 0
	m.When("Update", mock.Any, mock.Any).
		Call(func(root primitives.Sha256, diff merkle.TrieDiffs) (primitives.Sha256, error) {
			counter++
			return primitives.Sha256{counter}, nil
		}).
		AtLeast(0)
	return m
}

func (mm *MerkleMock) Update(rootMerkle primitives.Sha256, diffs merkle.TrieDiffs) (primitives.Sha256, error) {
	ret := mm.Mock.Called(rootMerkle, diffs)
	return ret.Get(0).(primitives.Sha256), ret.Error(1)
}
func (mm *MerkleMock) Forget(rootHash primitives.Sha256) {
	mm.Mock.Called(rootHash)
}

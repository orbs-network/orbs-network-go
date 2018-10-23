package statestorage

import (
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestWriteAtHeight(t *testing.T) {
	persistenceMock := statePersistenceMockWithWriteAnyNoErrors(0)
	d := newDriver(persistenceMock, 5)
	persistenceMock.
		When("Read", primitives.ContractName("c"), "k1").
		Return((*protocol.StateRecord)(nil), false, nil).
		Times(1)

	d.write(1, "c", "k1", "v1")

	v, exists, err := d.read(0, "c", "k1")
	require.NoError(t, err)
	require.EqualValues(t, false, exists)

	v, exists, err = d.read(1, "c", "k1")
	require.NoError(t, err)
	require.EqualValues(t, true, exists)
	require.EqualValues(t, "v1", v)

	// checking a future height is still legal.
	v, exists, err = d.read(200, "c", "k1")
	require.NoError(t, err)
	require.EqualValues(t, true, exists)
	require.EqualValues(t, "v1", v)

	ok, errCalled := persistenceMock.Verify()
	require.True(t, ok, "persistence mock called incorrectly")
	require.NoError(t, errCalled, "error happened when it should not")

}

func TestNoLayers(t *testing.T) {
	persistenceMock := &StatePersistenceMock{}
	persistenceMock.
		When("Write", mock.Any, mock.Any, mock.Any, mock.Any).
		Return(nil).
		Times(2)
	d := newDriver(persistenceMock, 0)
	d.writeFull(1, 1, primitives.MerkleSha256{1}, "c", "k", "v1")
	d.writeFull(2, 2, primitives.MerkleSha256{2}, "c", "k", "v2")

	_, _, err := d.read(1, "c", "k")
	require.EqualError(t, err, "requested height 1 is too old. oldest available block height is 2")

	ok, errCalled := persistenceMock.Verify()
	require.True(t, ok, "persistence mock called incorrectly")
	require.NoError(t, errCalled, "error happened when it should not")

}

func TestWriteAtHeightAndDeleteAtLaterHeight(t *testing.T) {
	d := newDriver(statePersistenceMockWithWriteAnyNoErrors(0), 5)
	d.write(1, "", "k1", "v1")
	d.write(2, "", "k1", "")

	v, exists, err := d.read(1, "", "k1")
	require.NoError(t, err)
	require.EqualValues(t, true, exists)
	require.EqualValues(t, "v1", v)

	v, exists, err = d.read(2, "", "k1")
	require.NoError(t, err)
	require.EqualValues(t, false, exists)
	require.EqualValues(t, "", v)
}

func TestMergeToPersistence(t *testing.T) {
	var writeCallCount byte = 1
	persistenceMock := &StatePersistenceMock{}
	persistenceMock.
		When("Write", mock.Any, mock.Any, mock.Any, mock.Any).
		Call(func(height primitives.BlockHeight, ts primitives.TimestampNano, root primitives.MerkleSha256, diff adapter.ChainState) error {
			expectedValue := fmt.Sprintf("v%v", writeCallCount)
			v := string(diff["c"]["k"].Value())
			require.EqualValues(t, expectedValue, v)
			require.EqualValues(t, writeCallCount, height)
			require.EqualValues(t, writeCallCount, ts)
			require.EqualValues(t, primitives.MerkleSha256{writeCallCount}, root)
			writeCallCount++
			return nil
		}).
		Times(2)
	d := newDriver(persistenceMock, 2)
	d.writeFull(1, 1, primitives.MerkleSha256{1}, "c", "k", "v1")
	d.writeFull(2, 2, primitives.MerkleSha256{2}, "c", "k", "v2")
	d.writeFull(3, 3, primitives.MerkleSha256{3}, "c", "k", "v3")
	d.writeFull(4, 4, primitives.MerkleSha256{4}, "c", "k", "v4")

	ok, errCalled := persistenceMock.Verify()
	require.True(t, ok, "persistence mock called incorrectly")
	require.NoError(t, errCalled, "error happened when it should not")
}

func TestReadOutOfRange(t *testing.T) {
	persistenceMock := statePersistenceMockWithWriteAnyNoErrors(2)
	d := newDriver(persistenceMock, 2)
	d.writeFull(1, 1, primitives.MerkleSha256{1}, "c", "k", "v1")
	d.writeFull(2, 2, primitives.MerkleSha256{2}, "c", "k", "v2")
	d.writeFull(3, 3, primitives.MerkleSha256{3}, "c", "k", "v3")
	d.writeFull(4, 4, primitives.MerkleSha256{4}, "c", "k", "v4")

	_, _, err := d.read(1, "c", "k")
	require.EqualError(t, err, "requested height 1 is too old. oldest available block height is 2")

	_, err = d.readHash(1)
	require.EqualError(t, err, "could not locate merkle hash for height 1. oldest available block height is 2")

	ok, errCalled := persistenceMock.Verify()
	require.True(t, ok, "persistence mock called incorrectly")
	require.NoError(t, errCalled, "error happened when it should not")
}

func TestReadHash(t *testing.T) {
	persistenceMock := statePersistenceMockWithWriteAnyNoErrors(1)
	d := newDriver(persistenceMock, 1)
	d.writeFull(1, 1, primitives.MerkleSha256{1}, "c", "k", "v1")
	d.writeFull(2, 2, primitives.MerkleSha256{2}, "c", "k", "v2")

	root, err := d.readHash(1)
	require.NoError(t, err)
	require.Equal(t, primitives.MerkleSha256{1}, root)

	root, err = d.readHash(2)
	require.NoError(t, err)
	require.Equal(t, primitives.MerkleSha256{2}, root)

	_, err = d.readHash(3)
	require.Error(t, err)

	ok, errCalled := persistenceMock.Verify()
	require.True(t, ok, "persistence mock called incorrectly")
	require.NoError(t, errCalled, "error happened when it should not")
}

type driver struct {
	inner *rollingRevisions
}

func newDriver(persistence adapter.StatePersistence, layers int) *driver {
	return &driver{
		newRollingRevisions(persistence, layers),
	}
}

func (d *driver) write(h primitives.BlockHeight, contract primitives.ContractName, kv ...string) error {
	diff := adapter.ChainState{contract: make(adapter.ContractState)}
	for i := 0; i < len(kv); i += 2 {
		diff[contract][kv[i]] = (&protocol.StateRecordBuilder{Key: []byte(kv[i]), Value: []byte(kv[i+1])}).Build()
	}
	return d.inner.addRevision(h, 0, primitives.MerkleSha256{}, diff)
}

func (d *driver) writeFull(h primitives.BlockHeight, ts primitives.TimestampNano, root primitives.MerkleSha256, contract primitives.ContractName, kv ...string) error {
	diff := adapter.ChainState{contract: make(adapter.ContractState)}
	for i := 0; i < len(kv); i += 2 {
		diff[contract][kv[i]] = (&protocol.StateRecordBuilder{Key: []byte(kv[i]), Value: []byte(kv[i+1])}).Build()
	}
	return d.inner.addRevision(h, ts, root, diff)
}

func (d *driver) read(h primitives.BlockHeight, contract primitives.ContractName, key string) (string, bool, error) {
	r, exists, err := d.inner.getRevisionRecord(h, contract, key)
	value := ""
	if r != nil {
		value = string(r.Value())
	}
	return value, exists, err
}

func (d *driver) readHash(h primitives.BlockHeight) (primitives.MerkleSha256, error) {
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

func (spm *StatePersistenceMock) Write(height primitives.BlockHeight, ts primitives.TimestampNano, root primitives.MerkleSha256, diff adapter.ChainState) error {
	return spm.Mock.Called(height, ts, root, diff).Error(0)
}
func (spm *StatePersistenceMock) Read(contract primitives.ContractName, key string) (*protocol.StateRecord, bool, error) {
	ret := spm.Mock.Called(contract, key)
	return ret.Get(0).(*protocol.StateRecord), ret.Bool(1), ret.Error(2)
}
func (spm *StatePersistenceMock) ReadMetadata() (primitives.BlockHeight, primitives.TimestampNano, primitives.MerkleSha256, error) {
	return 0, 0, primitives.MerkleSha256{}, nil
}

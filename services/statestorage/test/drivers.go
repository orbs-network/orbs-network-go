package test

import (
	"bytes"
	"fmt"
	"github.com/orbs-network/orbs-network-go/services/statestorage"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type driver struct {
	service     services.StateStorage
	persistence adapter.StatePersistence
	history 	driverConfig
}

type keyValue struct {
	key   string
	value []byte
}

func newStateStorageDriver(history int) *driver {
	if history <= 0 {
		history = 1
	}
	historySize := driverConfig{history}

	p := adapter.NewInMemoryStatePersistence()

	return &driver{persistence: p, service: statestorage.NewStateStorage(&historySize, p), history : historySize}
}

func (d *driver) readSingleKey(contract string, key string) ([]byte, error) {
	return d.readSingleKeyFromHistory(1, contract, key)
}

func (d *driver) readSingleKeyFromHistory(history int, contract string, key string)  ([]byte, error) {
	out, err := d.readKeysFromHistory(history, contract, key)
	if err != nil {
		return nil, err
	}
	return out[0].value, nil
}

func (d *driver) readKeys(contract string, keys ...string) ([]*keyValue, error) {
	return d.readKeysFromHistory(1, contract, keys...)
}

func (d *driver) readKeysFromHistory(history int, contract string, keys ...string) ([]*keyValue, error) {
	ripmdKeys := make([]primitives.Ripmd160Sha256, 0, len(keys))
	for _, key := range keys {
		ripmdKeys = append(ripmdKeys, primitives.Ripmd160Sha256(key))
	}
	out, err := d.service.ReadKeys(&services.ReadKeysInput{BlockHeight: primitives.BlockHeight(history), ContractName: primitives.ContractName(contract), Keys: ripmdKeys})

	if err != nil {
		return nil, err
	}

	if l, k := len(out.StateRecords), len(keys); l != k {
		panic(fmt.Sprintf("expected exactly %v elements in array. found %v", k, l))
	}

	result := make([]*keyValue, 0, len(keys))
	for i := range out.StateRecords {
		if actual, expected := out.StateRecords[i].Key(), keys[i]; !bytes.Equal(actual, []byte(expected)) {
			panic(fmt.Sprintf("expected output key %s to match input key %s", actual, expected))
		}
		result = append(result, &keyValue{string(out.StateRecords[i].Key()), out.StateRecords[i].Value()})
	}

	return result, nil
}

func (d *driver) write(contract string, key string, value []byte) {
	d.persistence.WriteState(1, primitives.ContractName(contract), (&protocol.StateRecordBuilder{Key: []byte(key), Value: value}).Build())
}

func (d *driver) writeSToBlockHeight(history int, contract string, key string, value []byte) {
	d.persistence.WriteState(primitives.BlockHeight(history), primitives.ContractName(contract), (&protocol.StateRecordBuilder{Key: []byte(key), Value: value}).Build())
}

func (d *driver) getBlockHeightAndTimestamp() (int, int, error) {
	output, err := d.service.GetStateStorageBlockHeight(&services.GetStateStorageBlockHeightInput{})
	return int(output.LastCommittedBlockHeight), int(output.LastCommittedBlockTimestamp), err
}

type driverConfig struct {
	historySize 	int
}

func (d *driverConfig) GetMaxStateHistory() uint64 {
	return uint64(d.historySize)
}

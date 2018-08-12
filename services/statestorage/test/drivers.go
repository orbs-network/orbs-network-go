package test

import (
	"bytes"
	"fmt"
	"github.com/orbs-network/orbs-network-go/services/statestorage"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type driver struct {
	service services.StateStorage
	history driverConfig
}

type keyValue struct {
	key   string
	value []byte
}

func newStateStorageDriver(history int) *driver {
	return newStateStorageDriverWithGrace(history, 0, 0)
}

func newStateStorageDriverWithGrace(history int, graceBlockDiff int, graceTimeoutMillis int) *driver {
	if history <= 0 {
		history = 1
	}
	historySize := driverConfig{
		history,
		graceBlockDiff,
		graceTimeoutMillis,
	}

	p := adapter.NewInMemoryStatePersistence()

	return &driver{service: statestorage.NewStateStorage(&historySize, p), history: historySize}
}

func (d *driver) readSingleKey(contract string, key string) ([]byte, error) {
	h, _, _ := d.getBlockHeightAndTimestamp()
	return d.readSingleKeyFromHistory(h, contract, key)
}

func (d *driver) readSingleKeyFromHistory(history int, contract string, key string) ([]byte, error) {
	out, err := d.readKeysFromHistory(history, contract, key)
	if err != nil {
		return nil, err
	}
	return out[0].value, nil
}

func (d *driver) readKeys(contract string, keys ...string) ([]*keyValue, error) {
	h, _, _ := d.getBlockHeightAndTimestamp()
	return d.readKeysFromHistory(h, contract, keys...)
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

func (d *driver) getBlockHeightAndTimestamp() (int, int, error) {
	output, err := d.service.GetStateStorageBlockHeight(&services.GetStateStorageBlockHeightInput{})
	return int(output.LastCommittedBlockHeight), int(output.LastCommittedBlockTimestamp), err
}

func (d *driver) commitStateDiff(state *services.CommitStateDiffInput) {
	d.service.CommitStateDiff(state)
}

func (d *driver) commitValuePairs(contract string, keyValues ...string) {
	h, _, _ := d.getBlockHeightAndTimestamp()
	d.commitValuePairsAtHeight(h+1, contract, keyValues...)
}

func (d *driver) commitValuePairsAtHeight(h int, contract string, keyValues ...string) {
	if len(keyValues)%2 != 0 {
		panic("expecting an array of key value pairs")
	}
	b := builders.ContractStateDiff().WithContractName(contract)

	for i := 0; i < len(keyValues); i += 2 {
		b.WithStringRecord(keyValues[i], keyValues[i+1])
	}

	contractStateDiff := b.Build()
	d.commitStateDiff(CommitStateDiff().WithBlockHeight(int(h)).WithDiff(contractStateDiff).Build())
}

type driverConfig struct {
	historySize                 int
	querySyncGraceBlockDist     int
	querySyncGraceTimeoutMillis int
}

func (d *driverConfig) StateHistoryRetentionInBlockHeights() uint64 {
	return uint64(d.historySize)
}

func (d *driverConfig) QuerySyncGraceBlockDist() uint64 {
	return uint64(d.querySyncGraceBlockDist)
}

func (d *driverConfig) QuerySyncGraceTimeoutMillis() uint64 {
	return uint64(d.querySyncGraceTimeoutMillis)
}

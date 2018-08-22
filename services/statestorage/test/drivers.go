package test

import (
	"bytes"
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/statestorage"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"os"
)

type driver struct {
	service services.StateStorage
}

type keyValue struct {
	key   string
	value []byte
}

func newStateStorageDriver(numOfStateRevisionsToRetain uint16) *driver {
	return newStateStorageDriverWithGrace(numOfStateRevisionsToRetain, 0, 0)
}

func newStateStorageDriverWithGrace(numOfStateRevisionsToRetain uint16, graceBlockDiff uint16, graceTimeoutMillis uint64) *driver {
	if numOfStateRevisionsToRetain <= 0 {
		numOfStateRevisionsToRetain = 1
	}

	conf := config.NewStateStorageConfig(numOfStateRevisionsToRetain, graceBlockDiff, graceTimeoutMillis)

	p := adapter.NewInMemoryStatePersistence()
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))

	return &driver{service: statestorage.NewStateStorage(conf, p, logger)}
}

func (d *driver) readSingleKey(contract string, key string) ([]byte, error) {
	h, _, _ := d.getBlockHeightAndTimestamp()
	return d.readSingleKeyFromRevision(h, contract, key)
}

func (d *driver) readSingleKeyFromRevision(revision int, contract string, key string) ([]byte, error) {
	out, err := d.readKeysFromRevision(revision, contract, key)
	if err != nil {
		return nil, err
	}
	return out[0].value, nil
}

func (d *driver) readKeys(contract string, keys ...string) ([]*keyValue, error) {
	h, _, _ := d.getBlockHeightAndTimestamp()
	return d.readKeysFromRevision(h, contract, keys...)
}

func (d *driver) readKeysFromRevision(revision int, contract string, keys ...string) ([]*keyValue, error) {
	ripmdKeys := make([]primitives.Ripmd160Sha256, 0, len(keys))
	for _, key := range keys {
		ripmdKeys = append(ripmdKeys, primitives.Ripmd160Sha256(key))
	}
	out, err := d.service.ReadKeys(&services.ReadKeysInput{BlockHeight: primitives.BlockHeight(revision), ContractName: primitives.ContractName(contract), Keys: ripmdKeys})

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

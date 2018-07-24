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
}

func newStateStorageDriver() *driver {
	p := adapter.NewInMemoryStatePersistence(&struct{}{})

	return &driver{persistence: p, service: statestorage.NewStateStorage(p)}
}

func (d *driver) readSingleKey(contract string, key string) ([]byte, error) {
	out, err := d.service.ReadKeys(&services.ReadKeysInput{ContractName: primitives.ContractName(contract), Keys: []primitives.Ripmd160Sha256{[]byte(key)}})

	if err != nil {
		return nil, err
	}

	if l := len(out.StateRecords); l != 1 {
		panic(fmt.Sprintf("expected exactly one element in array. found %v", l))
	}

	if actual, expected := out.StateRecords[0].Key(), []byte(key); !bytes.Equal(actual, expected) {
		panic(fmt.Sprintf("expected output key %s to match input key %s", actual, expected))
	}

	return out.StateRecords[0].Value(), nil
}

func (d *driver) write(contract string, key string, value []byte) {
	d.persistence.WriteState(primitives.ContractName(contract), (&protocol.StateRecordBuilder{Key: []byte(key), Value: value}).Build())
}

func (d *driver) getBlockHeightAndTimestamp() (int, int, error) {
	output, err := d.service.GetStateStorageBlockHeight(&services.GetStateStorageBlockHeightInput{})
	return int(output.LastCommittedBlockHeight), int(output.LastCommittedBlockTimestamp), err
}

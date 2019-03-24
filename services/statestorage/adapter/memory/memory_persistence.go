// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package memory

import (
	"bytes"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/merkle"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"sort"
	"strings"
	"sync"
)

type metrics struct {
	numberOfKeys      *metric.Gauge
	numberOfContracts *metric.Gauge
}

func newMetrics(m metric.Factory) *metrics {
	return &metrics{
		numberOfKeys:      m.NewGauge("StateStoragePersistence.TotalNumberOfKeys.Count"),
		numberOfContracts: m.NewGauge("StateStoragePersistence.TotalNumberOfContracts.Count"),
	}
}

type InMemoryStatePersistence struct {
	metrics    *metrics
	mutex      sync.RWMutex
	fullState  adapter.ChainState
	height     primitives.BlockHeight
	ts         primitives.TimestampNano
	merkleRoot primitives.Sha256
}

func NewStatePersistence(metricFactory metric.Factory) *InMemoryStatePersistence {
	_, merkleRoot := merkle.NewForest()
	// TODO(https://github.com/orbs-network/orbs-network-go/issues/582) - this is our hard coded Genesis block (height 0). Move this to a more dignified place or load from a file
	return &InMemoryStatePersistence{
		metrics:    newMetrics(metricFactory),
		mutex:      sync.RWMutex{},
		fullState:  adapter.ChainState{},
		height:     0,
		ts:         0,
		merkleRoot: merkleRoot,
	}
}

func (sp *InMemoryStatePersistence) reportSize() {
	nContracts := 0
	nKeys := 0
	for _, records := range sp.fullState {
		nContracts++
		nKeys = nKeys + len(records)
	}
	sp.metrics.numberOfKeys.Update(int64(nKeys))
	sp.metrics.numberOfContracts.Update(int64(nContracts))
}

func (sp *InMemoryStatePersistence) Write(height primitives.BlockHeight, ts primitives.TimestampNano, root primitives.Sha256, diff adapter.ChainState) error {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	sp.height = height
	sp.merkleRoot = root

	for contract, records := range diff {
		for _, record := range records {
			sp._writeOneRecord(primitives.ContractName(contract), record)
		}
	}
	sp.reportSize()
	return nil
}

func (sp *InMemoryStatePersistence) _writeOneRecord(c primitives.ContractName, r *protocol.StateRecord) {
	if _, ok := sp.fullState[c]; !ok {
		sp.fullState[c] = map[string]*protocol.StateRecord{}
	}

	if isZeroValue(r.Value()) {
		delete(sp.fullState[c], string(r.Key()))
		return
	}

	sp.fullState[c][string(r.Key())] = r
}

func (sp *InMemoryStatePersistence) Read(contract primitives.ContractName, key string) (*protocol.StateRecord, bool, error) {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()

	record, ok := sp.fullState[contract][key]
	return record, ok, nil
}

func (sp *InMemoryStatePersistence) ReadMetadata() (primitives.BlockHeight, primitives.TimestampNano, primitives.Sha256, error) {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()

	return sp.height, sp.ts, sp.merkleRoot, nil
}

func (sp *InMemoryStatePersistence) Dump() string {
	output := strings.Builder{}
	output.WriteString("{")
	output.WriteString(fmt.Sprintf("height: %v, data: {", sp.height))
	contracts := make([]primitives.ContractName, 0, len(sp.fullState))
	for c := range sp.fullState {
		contracts = append(contracts, c)
	}
	sort.Slice(contracts, func(i, j int) bool { return contracts[i] < contracts[j] })
	for _, currentContract := range contracts {
		keys := make([]string, 0, len(sp.fullState[currentContract]))
		for k := range sp.fullState[currentContract] {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

		output.WriteString(string(currentContract) + ":{")
		for _, k := range keys {
			output.WriteString(sp.fullState[currentContract][k].StringKey())
			output.WriteString(":")
			output.WriteString(sp.fullState[currentContract][k].StringValue())
			output.WriteString(",")
		}
		output.WriteString("},")
	}
	output.WriteString("}}")
	return output.String()
}

func isZeroValue(value []byte) bool {
	return bytes.Equal(value, []byte{})
}

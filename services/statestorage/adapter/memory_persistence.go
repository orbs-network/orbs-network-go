package adapter

import (
	"bytes"
	"fmt"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"sort"
	"strings"
	"sync"
)

type InMemoryStatePersistence struct {
	mutex      sync.RWMutex
	fullState  ChainState
	height     primitives.BlockHeight
	ts         primitives.TimestampNano
	merkleRoot primitives.MerkleSha256
}

func NewInMemoryStatePersistence() *InMemoryStatePersistence {

	// TODO - this is our hard coded Genesis block (height 0). Move this to a more dignified place or load from a file
	return &InMemoryStatePersistence{
		mutex:      sync.RWMutex{},
		fullState:  ChainState{},
		height:     0,
		ts:         0,
		merkleRoot: nil,
	}
}

func (sp *InMemoryStatePersistence) Write(height primitives.BlockHeight, ts primitives.TimestampNano, root primitives.MerkleSha256, diff ChainState) error {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	sp.height = height
	sp.merkleRoot = root

	for contract, records := range diff {
		for _, record := range records {
			sp._writeOneRecord(primitives.ContractName(contract), record)
		}
	}
	return nil
}

func (sp *InMemoryStatePersistence) _writeOneRecord(c primitives.ContractName, r *protocol.StateRecord) {
	if _, ok := sp.fullState[c]; !ok {
		sp.fullState[c] = map[string]*protocol.StateRecord{}
	}

	if isZeroValue(r.Value()) {
		delete(sp.fullState[c], r.Key().KeyForMap())
		return
	}

	sp.fullState[c][r.Key().KeyForMap()] = r
}

func (sp *InMemoryStatePersistence) Read(contract primitives.ContractName, key string) (*protocol.StateRecord, bool, error) {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()

	record, ok := sp.fullState[contract][key]
	return record, ok, nil
}

func (sp *InMemoryStatePersistence) ReadMetadata() (primitives.BlockHeight, primitives.TimestampNano, primitives.MerkleSha256, error) {
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

func (sp *InMemoryStatePersistence) Each(callback func (contract primitives.ContractName, record *protocol.StateRecord)) {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()

	for contract := range sp.fullState {
		for key := range sp.fullState[contract] {
			callback(contract, sp.fullState[contract][key])
		}
	}
}

// TODO - there is an identical method in statestorage. extract and reuse?
func isZeroValue(value []byte) bool {
	return bytes.Equal(value, []byte{})
}

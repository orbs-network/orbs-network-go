package adapter

import (
	"bytes"
	"fmt"
	"github.com/orbs-network/orbs-network-go/services/statestorage/merkle"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"sort"
	"strings"
	"sync"
)

type InMemoryStatePersistence struct {
	mutex       sync.RWMutex
	bState      ChainState
	bHeight     primitives.BlockHeight
	bTimestamp  primitives.TimestampNano
	bMerkleRoot primitives.MerkleSha256
}

func NewInMemoryStatePersistence() *InMemoryStatePersistence {

	// TODO - this is our hard coded Genesis block. Move this to a more dignified place or load from a file

	_, root := merkle.NewForest()
	return &InMemoryStatePersistence{
		mutex:       sync.RWMutex{},
		bState:      ChainState{},
		bHeight:     0,
		bTimestamp:  0,
		bMerkleRoot: root,
	}
}

func (sp *InMemoryStatePersistence) Write(height primitives.BlockHeight, ts primitives.TimestampNano, root primitives.MerkleSha256, diff ChainState) error {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	sp.bHeight = height
	sp.bMerkleRoot = root

	for contract, records := range diff {
		for _, record := range records {
			sp._writeOneRecord(primitives.ContractName(contract), record)
		}
	}
	return nil
}

func (sp *InMemoryStatePersistence) _writeOneRecord(c primitives.ContractName, r *protocol.StateRecord) {
	if _, ok := sp.bState[c]; !ok {
		sp.bState[c] = map[string]*protocol.StateRecord{}
	}

	if isZeroValue(r.Value()) {
		delete(sp.bState[c], r.Key().KeyForMap())
		return
	}

	sp.bState[c][r.Key().KeyForMap()] = r
}

func (sp *InMemoryStatePersistence) Read(contract primitives.ContractName, key string) (*protocol.StateRecord, bool, error) {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()

	record, ok := sp.bState[contract][key]
	return record, ok, nil
}

func (sp *InMemoryStatePersistence) ReadMetadata() (primitives.BlockHeight, primitives.TimestampNano, primitives.MerkleSha256, error) {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()

	return sp.bHeight, sp.bTimestamp, sp.bMerkleRoot, nil
}

func (sp *InMemoryStatePersistence) Dump() string {
	output := strings.Builder{}
	output.WriteString("{")
	output.WriteString(fmt.Sprintf("height: %v, data: {", sp.bHeight))
	contracts := make([]primitives.ContractName, 0, len(sp.bState))
	for c := range sp.bState {
		contracts = append(contracts, c)
	}
	sort.Slice(contracts, func(i, j int) bool { return contracts[i] < contracts[j] })
	for _, currentContract := range contracts {
		keys := make([]string, 0, len(sp.bState[currentContract]))
		for k := range sp.bState[currentContract] {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

		output.WriteString(string(currentContract) + ":{")
		for _, k := range keys {
			output.WriteString(sp.bState[currentContract][k].StringKey())
			output.WriteString(":")
			output.WriteString(sp.bState[currentContract][k].StringValue())
			output.WriteString(",")
		}
		output.WriteString("},")
	}
	output.WriteString("}}")
	return output.String()
}

// TODO - there is an identical method in statestorage. extract and reuse?
func isZeroValue(value []byte) bool {
	return bytes.Equal(value, []byte{})
}

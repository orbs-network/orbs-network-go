package adapter

import (
	"bytes"
	"fmt"
	"github.com/orbs-network/orbs-network-go/services/statestorage/merkle"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"sort"
	"strings"
	"sync"
)

type InMemoryStatePersistence struct {
	mu          sync.RWMutex
	snapshot    ChainDiff
	blockHeight primitives.BlockHeight
	timestamp   primitives.TimestampNano
	merkleRoot  primitives.MerkleSha256
}

func NewInMemoryStatePersistence() *InMemoryStatePersistence {

	// TODO - this is our hard coded Genesis block. Move this to a more dignified place or load from a file

	_, root := merkle.NewForest()
	return &InMemoryStatePersistence{
		mu:          sync.RWMutex{},
		snapshot:    ChainDiff{},
		blockHeight: 0,
		timestamp:   0,
		merkleRoot:  root,
	}
}

func (sp *InMemoryStatePersistence) WriteState(height primitives.BlockHeight, ts primitives.TimestampNano, root primitives.MerkleSha256, diff ChainDiff) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	sp.blockHeight = height
	sp.merkleRoot = root

	for contract, records := range diff {
		for _, record := range records {
			sp._writeOneRecord(height, primitives.ContractName(contract), record)
		}
	}
	return nil
}

func (sp *InMemoryStatePersistence) _writeOneRecord(h primitives.BlockHeight, c primitives.ContractName, r *protocol.StateRecord) {
	if _, ok := sp.snapshot[c]; !ok {
		sp.snapshot[c] = map[string]*protocol.StateRecord{}
	}

	if isZeroValue(r.Value()) {
		delete(sp.snapshot[c], r.Key().KeyForMap())
		return
	}

	sp.snapshot[c][r.Key().KeyForMap()] = r
}

func (sp *InMemoryStatePersistence) ReadState(height primitives.BlockHeight, contract primitives.ContractName, key string) (*protocol.StateRecord, bool, error) {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	if height != sp.blockHeight {
		return nil, false, errors.Errorf("block height mismatch. requested height %v, found %v", height, sp.blockHeight)
	}
	record, ok := sp.snapshot[contract][key]
	return record, ok, nil
}

func (sp *InMemoryStatePersistence) ReadBlockHeight() (primitives.BlockHeight, error) {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	return sp.blockHeight, nil
}

func (sp *InMemoryStatePersistence) ReadBlockTimestamp() (primitives.TimestampNano, error) {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	return sp.timestamp, nil
}

func (sp *InMemoryStatePersistence) ReadMerkleRoot(height primitives.BlockHeight) (primitives.MerkleSha256, error) {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	if height != sp.blockHeight {
		return nil, errors.Errorf("block height mismatch. requested height %v, found %v", height, sp.blockHeight)
	}
	return sp.merkleRoot, nil
}

func (sp *InMemoryStatePersistence) Dump() string {
	output := strings.Builder{}
	output.WriteString("{")
	output.WriteString(fmt.Sprintf("height: %v, data: {", sp.blockHeight))
	contracts := make([]primitives.ContractName, 0, len(sp.snapshot))
	for c := range sp.snapshot {
		contracts = append(contracts, c)
	}
	sort.Slice(contracts, func(i, j int) bool { return contracts[i] < contracts[j] })
	for _, currentContract := range contracts {
		keys := make([]string, 0, len(sp.snapshot[currentContract]))
		for k := range sp.snapshot[currentContract] {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

		output.WriteString(string(currentContract) + ":{")
		for _, k := range keys {
			output.WriteString(sp.snapshot[currentContract][k].StringKey())
			output.WriteString(":")
			output.WriteString(sp.snapshot[currentContract][k].StringValue())
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

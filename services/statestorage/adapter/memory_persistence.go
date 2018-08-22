package adapter

import (
	"bytes"
	"fmt"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"sort"
	"strings"
	"time"
)

type ContractState map[string]*protocol.StateRecord
type StateVersion map[primitives.ContractName]ContractState

type InMemoryStatePersistence struct {
	snapshots            map[primitives.BlockHeight]StateVersion
	blockTrackerForTests *synchronization.BlockTracker
}

func NewInMemoryStatePersistence() *InMemoryStatePersistence {
	stateDiffsContract := map[primitives.ContractName]ContractState{primitives.ContractName("BenchmarkToken"): {}}

	return &InMemoryStatePersistence{
		// TODO remove init with a hard coded contract once deploy/provisioning of contracts exists
		snapshots:            map[primitives.BlockHeight]StateVersion{primitives.BlockHeight(0): stateDiffsContract},
		blockTrackerForTests: synchronization.NewBlockTracker(0, 64000, time.Duration(1*time.Hour)),
	}
}

func (sp *InMemoryStatePersistence) WriteState(height primitives.BlockHeight, contractStateDiffs []*protocol.ContractStateDiff) error {
	if _, ok := sp.snapshots[height]; !ok {
		sp.snapshots[height] = sp.cloneCurrentStateDiff(height)
	}

	for _, stateDiffs := range contractStateDiffs {
		for i := stateDiffs.StateDiffsIterator(); i.HasNext(); {
			sp.writeOneContract(height, stateDiffs.ContractName(), i.NextStateDiffs())
		}
	}

	sp.blockTrackerForTests.IncrementHeight()

	return nil
}

func (sp *InMemoryStatePersistence) writeOneContract(height primitives.BlockHeight, contract primitives.ContractName, stateDiff *protocol.StateRecord) {
	if _, ok := sp.snapshots[height][contract]; !ok {
		sp.snapshots[height][contract] = map[string]*protocol.StateRecord{}
	}

	if isZeroValue(stateDiff.Value()) {
		delete(sp.snapshots[height][contract], stateDiff.Key().KeyForMap())
		return
	}

	sp.snapshots[height][contract][stateDiff.Key().KeyForMap()] = stateDiff
}

func (sp *InMemoryStatePersistence) cloneCurrentStateDiff(height primitives.BlockHeight) StateVersion {
	prevHeight := height - primitives.BlockHeight(1)
	if _, ok := sp.snapshots[prevHeight]; !ok {
		panic("trying to commit blocks not in order")
	}

	newStore := StateVersion{}
	for contract, contractStore := range sp.snapshots[prevHeight] {
		newStateRecordStore := map[string]*protocol.StateRecord{}
		for k, v := range contractStore {
			newStateRecordStore[k] = v
			//newStateRecordStore[k] = (&protocol.StateRecordBuilder{Key: v.Key(), Value: v.Value()}).Build()
		}
		newStore[contract] = newStateRecordStore
	}
	return newStore
}

func (sp *InMemoryStatePersistence) ReadState(height primitives.BlockHeight, contract primitives.ContractName) (map[string]*protocol.StateRecord, error) {
	if stateAtHeight, ok := sp.snapshots[height]; ok {
		if contractStateDiff, ok := stateAtHeight[contract]; ok {
			return contractStateDiff, nil
		} else {
			return nil, errors.Errorf("contract %v does not exist", contract)
		}
	} else {
		return nil, errors.Errorf("block %v does not exist in snapshot history", height)
	}
}

func (sp *InMemoryStatePersistence) Dump() string {
	blockHeights := make([]primitives.BlockHeight, 0, len(sp.snapshots))
	for bh := range sp.snapshots {
		blockHeights = append(blockHeights, bh)
	}
	sort.Slice(blockHeights, func(i, j int) bool { return blockHeights[i] < blockHeights[j] })

	output := strings.Builder{}
	output.WriteString("{")
	for _, currentBlock := range blockHeights {
		output.WriteString(fmt.Sprintf("height_%v:{", currentBlock))
		contracts := make([]primitives.ContractName, 0, len(sp.snapshots[currentBlock]))
		for c := range sp.snapshots[currentBlock] {
			contracts = append(contracts, c)
		}
		sort.Slice(contracts, func(i, j int) bool { return contracts[i] < contracts[j] })
		for _, currentContract := range contracts {
			keys := make([]string, 0, len(sp.snapshots[currentBlock][currentContract]))
			for k := range sp.snapshots[currentBlock][currentContract] {
				keys = append(keys, k)
			}
			sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

			output.WriteString(string(currentContract) + ":{")
			for _, k := range keys {
				output.WriteString(sp.snapshots[currentBlock][currentContract][k].StringKey())
				output.WriteString(":")
				output.WriteString(sp.snapshots[currentBlock][currentContract][k].StringValue())
				output.WriteString(",")
			}
			output.WriteString("},")
		}
		output.WriteString("},")
	}
	output.WriteString("}")
	return output.String()
}

func (sp *InMemoryStatePersistence) WaitUntilCommittedBlockOfHeight(height primitives.BlockHeight) error {
	return sp.blockTrackerForTests.WaitForBlock(height)
}

func isZeroValue(value []byte) bool {
	return bytes.Equal(value, []byte{})
}

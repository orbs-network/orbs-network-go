package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

type InMemoryStatePersistence struct {
	stateWritten chan bool
	stateDiffs   map[primitives.BlockHeight]map[primitives.ContractName]map[string]*protocol.StateRecord
}

func NewInMemoryStatePersistence() StatePersistence {
	stateDiffsContract := map[primitives.ContractName]map[string]*protocol.StateRecord{primitives.ContractName("BenchmarkToken"): {}}

	return &InMemoryStatePersistence{
		// TODO remove init with a hard coded contract once deploy/provisioning of contracts exists
		stateDiffs:   map[primitives.BlockHeight]map[primitives.ContractName]map[string]*protocol.StateRecord{primitives.BlockHeight(0): stateDiffsContract},
		stateWritten: make(chan bool, 10),
	}
}

func (sp *InMemoryStatePersistence) WriteState(height primitives.BlockHeight, contract primitives.ContractName, stateDiff *protocol.StateRecord) error {
	if _, ok := sp.stateDiffs[height]; !ok {
		sp.stateDiffs[height] = sp.cloneCurrentStateDiff(height)
	}

	if _, ok := sp.stateDiffs[height][contract]; !ok {
		sp.stateDiffs[height][contract] = map[string]*protocol.StateRecord{}
	}

	sp.stateDiffs[height][contract][stateDiff.Key().KeyForMap()] = stateDiff

	sp.stateWritten <- true

	return nil
}

func (sp *InMemoryStatePersistence) cloneCurrentStateDiff(height primitives.BlockHeight) map[primitives.ContractName]map[string]*protocol.StateRecord {
	prevHeight := height - primitives.BlockHeight(1)
	if _, ok := sp.stateDiffs[prevHeight]; !ok {
		panic("trying to commit blocks not in order")
	}

	newStore := map[primitives.ContractName]map[string]*protocol.StateRecord{}
	for contract, contractStore := range sp.stateDiffs[prevHeight] {
		newStateRecordStore := map[string]*protocol.StateRecord{}
		for k, v := range contractStore {
			newStateRecordStore[k] = v
			//newStateRecordStore[k] = (&protocol.StateRecordBuilder{Key: v.Key(), Value: v.Value()}).Build()
		}
		newStore[contract] = newStateRecordStore
	}
	return newStore
}

/*
func (sp *InMemoryStatePersistence) clearOldStateDiffs(current) {
	if nToRemove := uint64(len(sp.stateDiffs)) - sp.maxHistory; nToRemove > 0 {
		currRemove := uint64(current) - sp.maxHistory
		for ; nToRemove > 0 && currRemove > 0 ; {
			if _, ok := sp.stateDiffs[primitives.BlockHeight(currRemove)]; ok {
				delete(sp.stateDiffs, primitives.BlockHeight(currRemove))
				nToRemove--
			}
		}
	}
}
*/

func (sp *InMemoryStatePersistence) ReadState(height primitives.BlockHeight, contract primitives.ContractName) (map[string]*protocol.StateRecord, error) {
	if stateAtHeight, ok := sp.stateDiffs[height]; ok {
		if contractStateDiff, ok := stateAtHeight[contract]; ok {
			return contractStateDiff, nil
		} else {
			return nil, errors.Errorf("contract %v does not exist", contract)
		}
	} else {
		return nil, errors.Errorf("block %v does not exist in snapshot history", height)
	}
}

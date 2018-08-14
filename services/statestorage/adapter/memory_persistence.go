package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

type ContractState map[string]*protocol.StateRecord
type StateVersion map[primitives.ContractName]ContractState

type InMemoryStatePersistence struct {
	stateWritten chan bool
	stateDiffs   map[primitives.BlockHeight]StateVersion
}

func NewInMemoryStatePersistence() StatePersistence {
	stateDiffsContract := map[primitives.ContractName]ContractState{primitives.ContractName("BenchmarkToken"): {}}

	return &InMemoryStatePersistence{
		// TODO remove init with a hard coded contract once deploy/provisioning of contracts exists
		stateDiffs:   map[primitives.BlockHeight]StateVersion{primitives.BlockHeight(0): stateDiffsContract},
		stateWritten: make(chan bool, 10),
	}
}

func (sp *InMemoryStatePersistence) WriteState(height primitives.BlockHeight, contractStateDiffs []*protocol.ContractStateDiff) error {
	if _, ok := sp.stateDiffs[height]; !ok {
		sp.stateDiffs[height] = sp.cloneCurrentStateDiff(height)
	}

	for _, stateDiffs := range contractStateDiffs {
		for i := stateDiffs.StateDiffsIterator(); i.HasNext(); {
			sp.writeOneContract(height, stateDiffs.ContractName(), i.NextStateDiffs())
		}
	}

	sp.stateWritten <- true

	return nil
}

func (sp *InMemoryStatePersistence) writeOneContract(height primitives.BlockHeight, contract primitives.ContractName, stateDiff *protocol.StateRecord) {
	if _, ok := sp.stateDiffs[height][contract]; !ok {
		sp.stateDiffs[height][contract] = map[string]*protocol.StateRecord{}
	}
	sp.stateDiffs[height][contract][stateDiff.Key().KeyForMap()] = stateDiff
}

func (sp *InMemoryStatePersistence) cloneCurrentStateDiff(height primitives.BlockHeight) StateVersion {
	prevHeight := height - primitives.BlockHeight(1)
	if _, ok := sp.stateDiffs[prevHeight]; !ok {
		panic("trying to commit blocks not in order")
	}

	newStore := StateVersion{}
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

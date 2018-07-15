package statestorage

import (
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type service struct {
	services.StateStorage
	persistence adapter.StatePersistence
}

func NewStateStorage(persistence adapter.StatePersistence) services.StateStorage {
	return &service{
		persistence: persistence,
	}
}

func (s *service) CommitStateDiff(input *services.CommitStateDiffInput) (*services.CommitStateDiffOutput, error) {
	for _, stateDiffs := range input.ContractStateDiffs {
		for i := stateDiffs.StateDiffsIterator(); i.HasNext(); {
			s.persistence.WriteState(i.NextStateDiffs())
		}
	}

	return nil, nil
}

func (s *service) ReadKeys(input *services.ReadKeysInput) (*services.ReadKeysOutput, error) {
	var state []*protocol.StateRecord
	for _, stateDiff := range s.persistence.ReadState() {
		state = append(state, stateDiff)

	}
	output := &services.ReadKeysOutput{StateRecords: state}
	return output, nil
}

func (s *service) GetStateStorageBlockHeight(input *services.GetStateStorageBlockHeightInput) (*services.GetStateStorageBlockHeightOutput, error) {
	panic("Not implemented")
}

func (s *service) GetStateHash(input *services.GetStateHashInput) (*services.GetStateHashOutput, error) {
	panic("Not implemented")
}

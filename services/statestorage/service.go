package statestorage

import (
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"fmt"
	"bytes"
)

type service struct {
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
			s.persistence.WriteState(stateDiffs.ContractName(), i.NextStateDiffs())
		}
	}

	return nil, nil
}

func (s *service) ReadKeys(input *services.ReadKeysInput) (*services.ReadKeysOutput, error) {
	if input.ContractName == "" {
		return nil, fmt.Errorf("missing contract name")
	}

	var state []*protocol.StateRecord
	for _, stateDiff := range s.persistence.ReadState(input.ContractName) {
		for _, key := range input.Keys {
			if bytes.Equal(key, stateDiff.Key()) {
				state = append(state, stateDiff)
			}
		}

	}
	output := &services.ReadKeysOutput{StateRecords: state}
	if len(output.StateRecords) == 0 {
		return output, fmt.Errorf("no value found for input key(s)")
	}
	return output, nil
}

func (s *service) GetStateStorageBlockHeight(input *services.GetStateStorageBlockHeightInput) (*services.GetStateStorageBlockHeightOutput, error) {
	panic("Not implemented")
}

func (s *service) GetStateHash(input *services.GetStateHashInput) (*services.GetStateHashOutput, error) {
	panic("Not implemented")
}

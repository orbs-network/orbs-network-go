package statestorage

import (
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"fmt"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type service struct {
	persistence        adapter.StatePersistence
	lastCommittedBlock primitives.BlockHeight
}

func NewStateStorage(persistence adapter.StatePersistence) services.StateStorage {
	return &service{
		persistence: persistence,
	}
}

func (s *service) CommitStateDiff(input *services.CommitStateDiffInput) (*services.CommitStateDiffOutput, error) {
	committedBlock := input.ResultsBlockHeader.BlockHeight()
	if lastCommittedBlock := s.lastCommittedBlock; lastCommittedBlock+1 != committedBlock {
		return &services.CommitStateDiffOutput{NextDesiredBlockHeight: lastCommittedBlock + 1}, nil
	}

	for _, stateDiffs := range input.ContractStateDiffs {
		for i := stateDiffs.StateDiffsIterator(); i.HasNext(); {
			s.persistence.WriteState(stateDiffs.ContractName(), i.NextStateDiffs())
		}
	}
	s.lastCommittedBlock = committedBlock

	return &services.CommitStateDiffOutput{NextDesiredBlockHeight: committedBlock + 1}, nil
}

func (s *service) ReadKeys(input *services.ReadKeysInput) (*services.ReadKeysOutput, error) {
	if input.ContractName == "" {
		return nil, fmt.Errorf("missing contract name")
	}

	records := make([]*protocol.StateRecord,0,len(input.Keys))
	for _, key:= range input.Keys {
		record, ok := s.persistence.ReadState(input.ContractName)[key.KeyForMap()]
		if ok {
			records = append(records, record)
		}
	}

	output := &services.ReadKeysOutput{StateRecords: records}
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

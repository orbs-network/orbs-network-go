package state

import "github.com/orbs-network/orbs-spec/types/go/services"

func NewStateStorage() services.StateStorage {
	return &stateStorage{}
}

type stateStorage struct {
}

func (s *stateStorage) CommitStateDiff(input *services.CommitStateDiffInput) (*services.CommitStateDiffOutput, error) {
	panic("Not implemented")
}

func (s *stateStorage) ReadKeys(input *services.ReadKeysInput) (*services.ReadKeysOutput, error) {
	panic("Not implemented")
}

func (s *stateStorage) GetStateStorageBlockHeight(input *services.GetStateStorageBlockHeightInput) (*services.GetStateStorageBlockHeightOutput, error) {
	panic("Not implemented")
}

func (s *stateStorage) GetStateHash(input *services.GetStateHashInput) (*services.GetStateHashOutput, error) {
	panic("Not implemented")
}

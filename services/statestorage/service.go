package statestorage

import "github.com/orbs-network/orbs-spec/types/go/services"

type service struct {
	services.StateStorage
}

func NewStateStorage() services.StateStorage {
	return &service{}
}

func (s *service) CommitStateDiff(input *services.CommitStateDiffInput) (*services.CommitStateDiffOutput, error) {
	panic("Not implemented")
}

func (s *service) ReadKeys(input *services.ReadKeysInput) (*services.ReadKeysOutput, error) {
	panic("Not implemented")
}

func (s *service) GetStateStorageBlockHeight(input *services.GetStateStorageBlockHeightInput) (*services.GetStateStorageBlockHeightOutput, error) {
	panic("Not implemented")
}

func (s *service) GetStateHash(input *services.GetStateHashInput) (*services.GetStateHashOutput, error) {
	panic("Not implemented")
}

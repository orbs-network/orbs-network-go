package virtualmachine

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

func (s *service) getRecentBlockHeight() (primitives.BlockHeight, primitives.TimestampNano, error) {
	output, err := s.stateStorage.GetStateStorageBlockHeight(&services.GetStateStorageBlockHeightInput{})
	if err != nil {
		return 0, 0, err
	}
	return output.LastCommittedBlockHeight, output.LastCommittedBlockTimestamp, nil
}

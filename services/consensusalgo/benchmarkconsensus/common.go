package benchmarkconsensus

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

func (s *service) lastCommittedBlockHeight() primitives.BlockHeight {
	if s.lastCommittedBlock == nil {
		return 0
	}
	return s.lastCommittedBlock.TransactionsBlock.Header.BlockHeight()
}

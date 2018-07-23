package benchmarkconsensus

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
)

func (s *service) consensusRoundRunLoop() {
	var activeBlock *protocol.BlockPairContainer
	var err error

	for {
		s.reporting.Infof("Entered consensus round, last committed block height is %d", s.lastCommittedBlockHeight())

		if activeBlock == nil {
			activeBlock, err = s.generateNewProposedBlock()
			if err != nil {
				s.reporting.Error(err)
				time.Sleep(1 * time.Second) // TODO: replace with a configuration
			}
		}

		s.reporting.Infof("%v", activeBlock)
	}
}

func (s *service) lastCommittedBlockHeight() primitives.BlockHeight {
	return 0
}

func (s *service) generateNewProposedBlock() (*protocol.BlockPairContainer, error) {
	_, err := s.consensusContext.RequestNewTransactionsBlock(&services.RequestNewTransactionsBlockInput{})
	if err != nil {
		return nil, err
	}
	return nil, nil
}

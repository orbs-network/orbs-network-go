package benchmarkconsensus

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
)

func (s *service) consensusRoundRunLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			s.reporting.Infof("Consensus round run loop terminating with context")
			return
		default:
			err := s.consensusRoundTick()
			if err != nil {
				s.reporting.Error(err)
				time.Sleep(1 * time.Second) // TODO: replace with a configuration
			}
		}
	}
}

func (s *service) consensusRoundTick() (err error) {
	s.reporting.Infof("Entered consensus round, last committed block height is %d", s.lastCommittedBlockHeight())
	if s.activeBlock == nil {
		s.activeBlock, err = s.generateNewProposedBlock()
		if err != nil {
			return err
		}
	}
	return nil
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

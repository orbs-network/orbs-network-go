package leanhelixconsensus

import (
	"context"
	"github.com/orbs-network/lean-helix-go"
	lhprimitives "github.com/orbs-network/lean-helix-go/primitives"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

func (s *service) HandleBlockConsensus(ctx context.Context, input *handlers.HandleBlockConsensusInput) (*handlers.HandleBlockConsensusOutput, error) {
	panic("implement validate block consensus (call the lib)")
}

func (s *service) HandleLeanHelixMessage(ctx context.Context, input *gossiptopics.LeanHelixInput) (*gossiptopics.EmptyOutput, error) {
	panic("implement me")
}

func (s *service) getLastCommittedBlock() (primitives.BlockHeight, *protocol.BlockPairContainer) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.lastCommittedBlockUnderMutex == nil {
		return 0, nil
	}
	return s.lastCommittedBlockUnderMutex.TransactionsBlock.Header.BlockHeight(), s.lastCommittedBlockUnderMutex
}

func (s *service) RequestNewBlock(ctx context.Context, blockHeight lhprimitives.BlockHeight) (leanhelix.Block, error) {

	_lastCommittedBlockHeight, _lastCommittedBlock := s.getLastCommittedBlock()
	s.logger.Info("generating new proposed block", log.BlockHeight(_lastCommittedBlockHeight+1))

	// get tx
	txOutput, err := s.consensusContext.RequestNewTransactionsBlock(ctx, &services.RequestNewTransactionsBlockInput{
		BlockHeight:   _lastCommittedBlockHeight + 1,
		PrevBlockHash: digest.CalcTransactionsBlockHash(_lastCommittedBlock.TransactionsBlock),
	})
	if err != nil {
		return nil, err
	}

	// get rx
	rxOutput, err := s.consensusContext.RequestNewResultsBlock(ctx, &services.RequestNewResultsBlockInput{
		BlockHeight:       _lastCommittedBlockHeight + 1,
		PrevBlockHash:     digest.CalcResultsBlockHash(_lastCommittedBlock.ResultsBlock),
		TransactionsBlock: txOutput.TransactionsBlock,
	})
	if err != nil {
		return nil, err
	}

	blockPairWrapper := NewBlockPairWrapper(&protocol.BlockPairContainer{
		TransactionsBlock: txOutput.TransactionsBlock,
		ResultsBlock:      rxOutput.ResultsBlock,
	})

	// generate signed block
	return blockPairWrapper, nil
}

func (s *service) CalculateBlockHash(block leanhelix.Block) lhprimitives.Uint256 {
	panic("implement me - call digest() ")
}

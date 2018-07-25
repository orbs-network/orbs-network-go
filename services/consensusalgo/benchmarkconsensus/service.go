package benchmarkconsensus

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

type Config interface {
	NetworkSize(asOfBlock uint64) uint32
	NodePublicKey() primitives.Ed25519Pkey
	ConstantConsensusLeader() primitives.Ed25519Pkey
	ActiveConsensusAlgo() consensus.ConsensusAlgoType
}

type service struct {
	gossip           gossiptopics.BenchmarkConsensus
	blockStorage     services.BlockStorage
	consensusContext services.ConsensusContext
	reporting        instrumentation.Reporting
	config           Config

	activeBlock        *protocol.BlockPairContainer
	lastCommittedBlock *protocol.BlockPairContainer
}

func NewBenchmarkConsensusAlgo(
	ctx context.Context,
	gossip gossiptopics.BenchmarkConsensus,
	blockStorage services.BlockStorage,
	consensusContext services.ConsensusContext,
	reporting instrumentation.Reporting,
	config Config,
) services.ConsensusAlgoBenchmark {

	s := &service{
		gossip:           gossip,
		blockStorage:     blockStorage,
		consensusContext: consensusContext,
		reporting:        reporting,
		config:           config,
	}

	gossip.RegisterBenchmarkConsensusHandler(s)
	blockStorage.RegisterConsensusBlocksHandler(s)
	if config.ActiveConsensusAlgo() == consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS && config.ConstantConsensusLeader().Equal(config.NodePublicKey()) {
		go s.consensusRoundRunLoop(ctx)
	}
	return s
}

func (s *service) OnNewConsensusRound(input *services.OnNewConsensusRoundInput) (*services.OnNewConsensusRoundOutput, error) {
	panic("Not implemented")
}

func (s *service) HandleTransactionsBlock(input *handlers.HandleTransactionsBlockInput) (*handlers.HandleTransactionsBlockOutput, error) {
	panic("Not implemented")
}

func (s *service) HandleResultsBlock(input *handlers.HandleResultsBlockInput) (*handlers.HandleResultsBlockOutput, error) {
	panic("Not implemented")
}

func (s *service) HandleBenchmarkConsensusCommit(input *gossiptopics.BenchmarkConsensusCommitInput) (*gossiptopics.EmptyOutput, error) {
	if input.Message == nil || input.Message.BlockPair == nil {
		panic("HandleBenchmarkConsensusCommit received corrupt args")
	}
	s.nonLeaderHandleCommit(input.Message.BlockPair)
	return nil, nil
}

func (s *service) HandleBenchmarkConsensusCommitted(input *gossiptopics.BenchmarkConsensusCommittedInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}

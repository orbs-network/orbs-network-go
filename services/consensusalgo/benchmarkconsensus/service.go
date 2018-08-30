package benchmarkconsensus

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"math"
	"sync"
	"time"
)

const blockHeightNone = primitives.BlockHeight(math.MaxUint64)

type Config interface {
	NodePublicKey() primitives.Ed25519PublicKey
	NodePrivateKey() primitives.Ed25519PrivateKey
	NetworkSize(asOfBlock uint64) uint32
	FederationNodes(asOfBlock uint64) map[string]config.FederationNode
	ConstantConsensusLeader() primitives.Ed25519PublicKey
	ActiveConsensusAlgo() consensus.ConsensusAlgoType
	BenchmarkConsensusRetryInterval() time.Duration
}

type service struct {
	gossip           gossiptopics.BenchmarkConsensus
	blockStorage     services.BlockStorage
	consensusContext services.ConsensusContext
	reporting        log.BasicLogger
	config           Config

	isLeader           bool
	mutex              *sync.Mutex
	lastCommittedBlock *protocol.BlockPairContainer

	// leader only
	lastSuccessfullyVotedBlock primitives.BlockHeight
	successfullyVotedBlocks    chan primitives.BlockHeight
	lastCommittedBlockVoters   map[string]bool
}

func NewBenchmarkConsensusAlgo(
	ctx context.Context,
	gossip gossiptopics.BenchmarkConsensus,
	blockStorage services.BlockStorage,
	consensusContext services.ConsensusContext,
	reporting log.BasicLogger,
	config Config,
) services.ConsensusAlgoBenchmark {

	s := &service{
		gossip:           gossip,
		blockStorage:     blockStorage,
		consensusContext: consensusContext,
		reporting:        reporting.For(log.Service("consensus-algo-benchmark")),
		config:           config,

		isLeader: config.ConstantConsensusLeader().Equal(config.NodePublicKey()),

		// leader only
		mutex: &sync.Mutex{},
		lastSuccessfullyVotedBlock: blockHeightNone,
		successfullyVotedBlocks:    make(chan primitives.BlockHeight),
		lastCommittedBlockVoters:   make(map[string]bool),
	}

	gossip.RegisterBenchmarkConsensusHandler(s)
	blockStorage.RegisterConsensusBlocksHandler(s)

	if config.ActiveConsensusAlgo() == consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS && s.isLeader {
		go s.leaderConsensusRoundRunLoop(ctx)
	}

	return s
}

func (s *service) HandleBlockConsensus(input *handlers.HandleBlockConsensusInput) (*handlers.HandleBlockConsensusOutput, error) {
	return nil, s.handleBlockConsensusFromHandler(input.Mode, input.BlockType, input.BlockPair, input.PrevCommittedBlockPair)
}

func (s *service) HandleBenchmarkConsensusCommit(input *gossiptopics.BenchmarkConsensusCommitInput) (*gossiptopics.EmptyOutput, error) {
	if !s.isLeader {
		s.nonLeaderHandleCommit(input.Message.BlockPair)
	}
	return nil, nil
}

func (s *service) HandleBenchmarkConsensusCommitted(input *gossiptopics.BenchmarkConsensusCommittedInput) (*gossiptopics.EmptyOutput, error) {
	if s.isLeader {
		s.leaderHandleCommittedVote(input.Message.Sender, input.Message.Status)
	}
	return nil, nil
}

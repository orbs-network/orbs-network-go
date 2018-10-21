package benchmarkconsensus

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
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

var LogTag = log.Service("consensus-algo-benchmark")

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
	logger           log.BasicLogger
	config           Config

	isLeader           bool
	mutex              *sync.Mutex
	lastCommittedBlock *protocol.BlockPairContainer

	// leader only
	lastSuccessfullyVotedBlock            primitives.BlockHeight
	successfullyVotedBlocks               chan primitives.BlockHeight
	lastCommittedBlockVoters              map[string]bool
	lastCommittedBlockVotersReachedQuorum bool

	metrics *metrics
}

type metrics struct {
	consensusRoundTick     *metric.Histogram
	failedConsensusTicks   *metric.Rate
	timedOutConsensusTicks *metric.Rate
	votingLatency          *metric.Histogram
}

func newMetrics(m metric.Factory, consensusTimeout time.Duration, collectVotesTimeout time.Duration) *metrics {
	return &metrics{
		consensusRoundTick:     m.NewLatency("ConsensusAlgo.Benchmark.RoundTick", consensusTimeout),
		failedConsensusTicks:   m.NewRate("ConsensusAlgo.Benchmark.FailedTicks"),
		timedOutConsensusTicks: m.NewRate("ConsensusAlgo.Benchmark.TimedOutTicks"),
		votingLatency:          m.NewLatency("ConsensusAlgo.Benchmark.VotingLatency", collectVotesTimeout),
	}
}

func NewBenchmarkConsensusAlgo(
	ctx context.Context,
	gossip gossiptopics.BenchmarkConsensus,
	blockStorage services.BlockStorage,
	consensusContext services.ConsensusContext,
	logger log.BasicLogger,
	config Config,
	metricFactory metric.Factory,
) services.ConsensusAlgoBenchmark {

	s := &service{
		gossip:           gossip,
		blockStorage:     blockStorage,
		consensusContext: consensusContext,
		logger:           logger.WithTags(LogTag),
		config:           config,

		isLeader: config.ConstantConsensusLeader().Equal(config.NodePublicKey()),

		// leader only
		mutex: &sync.Mutex{},
		lastSuccessfullyVotedBlock:            blockHeightNone,
		successfullyVotedBlocks:               make(chan primitives.BlockHeight),
		lastCommittedBlockVoters:              make(map[string]bool),
		lastCommittedBlockVotersReachedQuorum: false,

		metrics: newMetrics(metricFactory, config.BenchmarkConsensusRetryInterval(), config.BenchmarkConsensusRetryInterval()),
	}

	gossip.RegisterBenchmarkConsensusHandler(s)
	blockStorage.RegisterConsensusBlocksHandler(s)

	if config.ActiveConsensusAlgo() == consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS && s.isLeader {
		go s.leaderConsensusRoundRunLoop(ctx)
	}

	return s
}

func (s *service) HandleBlockConsensus(ctx context.Context, input *handlers.HandleBlockConsensusInput) (*handlers.HandleBlockConsensusOutput, error) {
	return nil, s.handleBlockConsensusFromHandler(input.Mode, input.BlockType, input.BlockPair, input.PrevCommittedBlockPair)
}

func (s *service) HandleBenchmarkConsensusCommit(ctx context.Context, input *gossiptopics.BenchmarkConsensusCommitInput) (*gossiptopics.EmptyOutput, error) {
	if !s.isLeader {
		s.nonLeaderHandleCommit(ctx, input.Message.BlockPair)
	}
	return nil, nil
}

func (s *service) HandleBenchmarkConsensusCommitted(ctx context.Context, input *gossiptopics.BenchmarkConsensusCommittedInput) (*gossiptopics.EmptyOutput, error) {
	if s.isLeader {
		s.leaderHandleCommittedVote(input.Message.Sender, input.Message.Status)
	}
	return nil, nil
}

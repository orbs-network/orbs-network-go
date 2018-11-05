package benchmarkconsensus

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
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
	ConsensusRequiredQuorumPercentage() uint32
}

type service struct {
	gossip           gossiptopics.BenchmarkConsensus
	blockStorage     services.BlockStorage
	consensusContext services.ConsensusContext
	logger           log.BasicLogger
	config           Config

	isLeader                bool
	successfullyVotedBlocks chan primitives.BlockHeight // leader only

	mutex                                           *sync.RWMutex
	lastCommittedBlockUnderMutex                    *protocol.BlockPairContainer
	lastSuccessfullyVotedBlock                      primitives.BlockHeight // leader only
	lastCommittedBlockVotersUnderMutex              map[string]bool        // leader only
	lastCommittedBlockVotersReachedQuorumUnderMutex bool                   // leader only

	metrics *metrics
}

type metrics struct {
	consensusRoundTickTime     *metric.Histogram
	failedConsensusTicksRate   *metric.Rate
	timedOutConsensusTicksRate *metric.Rate
	votingTime                 *metric.Histogram
}

func newMetrics(m metric.Factory, consensusTimeout time.Duration, collectVotesTimeout time.Duration) *metrics {
	return &metrics{
		consensusRoundTickTime:     m.NewLatency("ConsensusAlgo.Benchmark.RoundTickTime", consensusTimeout),
		votingTime:                 m.NewLatency("ConsensusAlgo.Benchmark.VotingTime", collectVotesTimeout),
		failedConsensusTicksRate:   m.NewRate("ConsensusAlgo.Benchmark.FailedTicksPerSecond"),
		timedOutConsensusTicksRate: m.NewRate("ConsensusAlgo.Benchmark.TimedOutTicksPerSecond"),
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

		isLeader:                   config.ConstantConsensusLeader().Equal(config.NodePublicKey()),
		successfullyVotedBlocks:    make(chan primitives.BlockHeight), // leader only
		lastSuccessfullyVotedBlock: blockHeightNone,                   // leader only

		mutex: &sync.RWMutex{},
		lastCommittedBlockVotersUnderMutex:              make(map[string]bool), // leader only
		lastCommittedBlockVotersReachedQuorumUnderMutex: false,                 // leader only

		metrics: newMetrics(metricFactory, config.BenchmarkConsensusRetryInterval(), config.BenchmarkConsensusRetryInterval()),
	}

	gossip.RegisterBenchmarkConsensusHandler(s)
	blockStorage.RegisterConsensusBlocksHandler(s)

	if config.ActiveConsensusAlgo() == consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS && s.isLeader {
		supervised.GoForever(ctx, logger, func() {
			s.leaderConsensusRoundRunLoop(ctx)
		})
	}

	return s
}

func (s *service) HandleBlockConsensus(ctx context.Context, input *handlers.HandleBlockConsensusInput) (*handlers.HandleBlockConsensusOutput, error) {
	return nil, s.handleBlockConsensusFromHandler(input.Mode, input.BlockType, input.BlockPair, input.PrevCommittedBlockPair)
}

func (s *service) HandleBenchmarkConsensusCommit(ctx context.Context, input *gossiptopics.BenchmarkConsensusCommitInput) (*gossiptopics.EmptyOutput, error) {
	if !s.isLeader {
		return nil, s.nonLeaderHandleCommit(ctx, input.Message.BlockPair)
	}
	return nil, nil
}

func (s *service) HandleBenchmarkConsensusCommitted(ctx context.Context, input *gossiptopics.BenchmarkConsensusCommittedInput) (*gossiptopics.EmptyOutput, error) {
	if s.isLeader {
		return nil, s.leaderHandleCommittedVote(input.Message.Sender, input.Message.Status)
	}
	return nil, nil
}

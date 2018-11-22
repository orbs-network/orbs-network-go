package leanhelixconsensus

import (
	"context"
	"github.com/orbs-network/lean-helix-go"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"sync"
	"time"
)

var LogTag = log.Service("consensus-algo-lean-helix")

type lastCommittedBlock struct {
	sync.RWMutex
	block *protocol.BlockPairContainer
}

type service struct {
	blockStorage     services.BlockStorage
	comm             *networkCommunication
	consensusContext services.ConsensusContext
	logger           log.BasicLogger
	config           Config
	metrics          *metrics
	leanHelix        leanhelix.LeanHelix
	*lastCommittedBlock
}

func (s *service) HandleLeanHelixMessage(ctx context.Context, input *gossiptopics.LeanHelixInput) (*gossiptopics.EmptyOutput, error) {
	return s.comm.HandleLeanHelixMessage(ctx, input)
}

type metrics struct {
	consensusRoundTickTime     *metric.Histogram
	failedConsensusTicksRate   *metric.Rate
	timedOutConsensusTicksRate *metric.Rate
	votingTime                 *metric.Histogram
}

type Config interface {
	NodePublicKey() primitives.Ed25519PublicKey
	NodePrivateKey() primitives.Ed25519PrivateKey

	LeanHelixConsensusRoundTimeoutInterval() time.Duration
	ActiveConsensusAlgo() consensus.ConsensusAlgoType
}

func newMetrics(m metric.Factory, consensusTimeout time.Duration) *metrics {
	return &metrics{
		consensusRoundTickTime:     m.NewLatency("ConsensusAlgo.LeanHelix.RoundTickTime", consensusTimeout),
		failedConsensusTicksRate:   m.NewRate("ConsensusAlgo.LeanHelix.FailedTicksPerSecond"),
		timedOutConsensusTicksRate: m.NewRate("ConsensusAlgo.LeanHelix.TimedOutTicksPerSecond"),
	}
}

func NewLeanHelixConsensusAlgo(
	ctx context.Context,
	gossip gossiptopics.LeanHelix,
	blockStorage services.BlockStorage,

	consensusContext services.ConsensusContext,
	logger log.BasicLogger,
	config Config,
	metricFactory metric.Factory,

) services.ConsensusAlgoLeanHelix {

	comm := NewNetworkCommunication(gossip)
	mgr := NewKeyManager(config.NodePublicKey(), config.NodePrivateKey())
	provider := NewBlockProvider(config.LeanHelixConsensusRoundTimeoutInterval(), config.NodePublicKey(), config.NodePrivateKey())
	electionTrigger := leanhelix.NewTimerBasedElectionTrigger(config.LeanHelixConsensusRoundTimeoutInterval())

	s := &service{
		blockStorage:     blockStorage,
		comm:             comm,
		consensusContext: consensusContext,
		logger:           logger.WithTags(LogTag),
		config:           config,
		metrics:          newMetrics(metricFactory, config.LeanHelixConsensusRoundTimeoutInterval()),
		leanHelix:        nil,
	}

	leanHelixConfig := &leanhelix.Config{
		NetworkCommunication: comm,
		BlockUtils:           provider,
		KeyManager:           mgr,
		ElectionTrigger:      electionTrigger,
	}

	leanHelix := leanhelix.NewLeanHelix(leanHelixConfig)

	s.leanHelix = leanHelix

	gossip.RegisterLeanHelixHandler(comm)

	// TODO uncomment after BlockStorage mutex issues (s.lastBlockLock) are fixed
	//blockStorage.RegisterConsensusBlocksHandler(s)

	if config.ActiveConsensusAlgo() == consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX {
		go s.leanHelix.Start(ctx, 1) // TODO Get the block height from someplace
	}

	return s
}

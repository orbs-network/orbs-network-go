package leanhelixconsensus

import (
	"context"
	"github.com/orbs-network/lean-helix-go"
	lhprimitives "github.com/orbs-network/lean-helix-go/primitives"
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

type service struct {
	gossip           gossiptopics.LeanHelix
	blockStorage     services.BlockStorage
	consensusContext services.ConsensusContext
	logger           log.BasicLogger
	config           Config
	metrics          *metrics
	leanHelix        leanhelix.LeanHelix

	lastCommittedBlock struct {
		sync.RWMutex
		block *protocol.BlockPairContainer
	}
	messageReceivers        map[int]func(ctx context.Context, message leanhelix.ConsensusRawMessage)
	messageReceiversCounter int
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

type BlockPairWrapper struct {
	blockPair *protocol.BlockPairContainer
}

func (b *BlockPairWrapper) Height() lhprimitives.BlockHeight {
	return lhprimitives.BlockHeight(b.blockPair.TransactionsBlock.Header.BlockHeight())
}

func (b *BlockPairWrapper) BlockHash() lhprimitives.Uint256 {
	// TODO This is surely incorrect, fix to use the right hash
	return lhprimitives.Uint256(b.blockPair.TransactionsBlock.Header.MetadataHash())
}

func NewBlockPairWrapper(blockPair *protocol.BlockPairContainer) *BlockPairWrapper {
	return &BlockPairWrapper{
		blockPair: blockPair,
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

	electionTrigger := leanhelix.NewTimerBasedElectionTrigger(config.LeanHelixConsensusRoundTimeoutInterval())

	s := &service{
		gossip:                  gossip,
		blockStorage:            blockStorage,
		consensusContext:        consensusContext,
		logger:                  logger.WithTags(LogTag),
		config:                  config,
		metrics:                 newMetrics(metricFactory, config.LeanHelixConsensusRoundTimeoutInterval()),
		leanHelix:               nil,
		messageReceivers:        make(map[int]func(ctx context.Context, message leanhelix.ConsensusRawMessage)),
		messageReceiversCounter: 0,
	}

	leanHelixConfig := &leanhelix.Config{
		NetworkCommunication: s,
		BlockUtils:           s,
		KeyManager:           s,
		ElectionTrigger:      electionTrigger,
	}

	leanHelix := leanhelix.NewLeanHelix(leanHelixConfig)

	s.leanHelix = leanHelix

	gossip.RegisterLeanHelixHandler(s)

	// TODO uncomment after BlockStorage mutex issues (s.lastBlockLock) are fixed
	//blockStorage.RegisterConsensusBlocksHandler(s)

	if config.ActiveConsensusAlgo() == consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX {
		go s.leanHelix.Start(ctx, 1) // TODO Get the block height from someplace
	}

	return s
}

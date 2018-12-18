package leanhelixconsensus

import (
	"context"
	"github.com/orbs-network/lean-helix-go"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"time"
)

var LogTag = log.Service("consensus-algo-lean-helix")

type service struct {
	blockStorage     services.BlockStorage
	membership       *membership
	comm             *communication
	consensusContext services.ConsensusContext
	logger           log.BasicLogger
	config           Config
	metrics          *metrics
	leanHelix        *leanhelix.LeanHelix
}

type metrics struct {
	consensusRoundTickTime     *metric.Histogram
	failedConsensusTicksRate   *metric.Rate
	timedOutConsensusTicksRate *metric.Rate
	votingTime                 *metric.Histogram
}

type Config interface {
	NodeAddress() primitives.NodeAddress
	NodePrivateKey() primitives.EcdsaSecp256K1PrivateKey
	FederationNodes(asOfBlock uint64) map[string]config.FederationNode
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
	parentLogger log.BasicLogger,
	config Config,
	metricFactory metric.Factory,

) services.ConsensusAlgoLeanHelix {

	logger := parentLogger.WithTags(LogTag)
	logger.Info("NewLeanHelixConsensusAlgo() start")
	comm := NewCommunication(logger, gossip)
	membership := NewMembership(logger, config.NodeAddress(), consensusContext)
	mgr := NewKeyManager(config.NodePrivateKey())
	genesisBlock := generateGenesisBlock(config.NodePrivateKey())
	blockHeight := lhprimitives.BlockHeight(genesisBlock.TransactionsBlock.Header.BlockHeight() + 1)

	provider := NewBlockProvider(logger, blockStorage, consensusContext, config.NodeAddress(), config.NodePrivateKey())

	// Configure to be ~5 times the minimum wait for transactions (consensus context)
	electionTrigger := leanhelix.NewTimerBasedElectionTrigger(config.LeanHelixConsensusRoundTimeoutInterval())

	s := &service{
		comm:         comm,
		blockStorage: blockStorage,
		logger:       logger,
		config:       config,
		metrics:      newMetrics(metricFactory, config.LeanHelixConsensusRoundTimeoutInterval()),
		leanHelix:    nil,
	}

	leanHelixConfig := &leanhelix.Config{
		Communication:   comm,
		Membership:      membership,
		BlockUtils:      provider,
		KeyManager:      mgr,
		ElectionTrigger: electionTrigger,
		Logger:          NewLoggerWrapper(parentLogger, true),
	}

	logger.Info("NewLeanHelixConsensusAlgo() calling NewLeanHelix()")
	onCommit := func(block leanhelix.Block) {
		parentLogger.Info("YEYYYY CONSENSUS!!!! will save to block storage", log.Stringable("block-height", block.Height()))
		blockPairWrapper := block.(*BlockPairWrapper)
		blockPair := blockPairWrapper.blockPair
		s.saveToBlockStorage(ctx, blockPair)
	}
	leanHelix := leanhelix.NewLeanHelix(leanHelixConfig, onCommit)

	s.leanHelix = leanHelix

	gossip.RegisterLeanHelixHandler(s)
	if config.ActiveConsensusAlgo() == consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX {
		parentLogger.Info("LeanHelix go routine starts", log.BlockHeight(primitives.BlockHeight(blockHeight)))

		supervised.GoForever(ctx, logger, func() {
			s.leanHelix.Run(ctx)
		})
		s.leanHelix.UpdateConsensusRound(ToBlockPairWrapper(genesisBlock))
		logger.Info("NewLeanHelixConsensusAlgo() Sent genesis block to AcknowledgeBlockConsensus()")

		blockStorage.RegisterConsensusBlocksHandler(s)
		logger.Info("NewLeanHelixConsensusAlgo() active algo", log.Stringable("active-consensus-algo", config.ActiveConsensusAlgo()))

	} else {
		parentLogger.Info("LeanHelix is not the active consensus algo, not starting its consensus loop")
	}
	return s
}

func (s *service) saveToBlockStorage(ctx context.Context, blockPair *protocol.BlockPairContainer) error {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	hash := digest.CalcTransactionsBlockHash(blockPair.TransactionsBlock)
	logger.Info("saving block to storage", log.Stringable("block-hash", hash), log.BlockHeight(blockPair.TransactionsBlock.Header.BlockHeight()))
	_, err := s.blockStorage.CommitBlock(ctx, &services.CommitBlockInput{
		BlockPair: blockPair,
	})
	return err
}

func (s *service) HandleBlockConsensus(ctx context.Context, input *handlers.HandleBlockConsensusInput) (*handlers.HandleBlockConsensusOutput, error) {

	blockType := input.BlockType
	mode := input.Mode
	blockPair := input.BlockPair
	prevCommittedBlockPair := input.PrevCommittedBlockPair
	if blockType != protocol.BLOCK_TYPE_BLOCK_PAIR {
		return nil, errors.Errorf("handler received unsupported block type %s", blockType)
	}

	// validate the block consensus
	if mode == handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE || mode == handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_ONLY {
		err := s.validateBlockConsensus(blockPair, prevCommittedBlockPair)
		if err != nil {
			return nil, err
		}
	}

	prevBlock := ToBlockPairWrapper(blockPair)
	s.leanHelix.UpdateConsensusRound(prevBlock)

	return nil, nil
}

func (s *service) HandleLeanHelixMessage(ctx context.Context, input *gossiptopics.LeanHelixInput) (*gossiptopics.EmptyOutput, error) {
	consensusRawMessage := &leanhelix.ConsensusRawMessage{
		Content: input.Message.Content,
		Block:   ToBlockPairWrapper(input.Message.BlockPair),
	}
	s.leanHelix.GossipMessageReceived(ctx, consensusRawMessage)
	return nil, nil
}

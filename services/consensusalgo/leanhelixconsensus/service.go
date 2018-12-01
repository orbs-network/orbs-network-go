package leanhelixconsensus

import (
	"context"
	"github.com/orbs-network/lean-helix-go"
	lhprimitives "github.com/orbs-network/lean-helix-go/primitives"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
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
	comm             *networkCommunication
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
	NodePublicKey() primitives.Ed25519PublicKey
	NodePrivateKey() primitives.Ed25519PrivateKey
	FederationNodes(asOfBlock uint64) map[string]config.FederationNode
	LeanHelixConsensusRoundTimeoutInterval() time.Duration
	ConsensusContextMinimalBlockTime() time.Duration
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
	comm := NewNetworkCommunication(logger, consensusContext, gossip)
	mgr := NewKeyManager(logger, config.NodePublicKey(), config.NodePrivateKey())
	genesisBlock := generateGenesisBlock(config.NodePrivateKey())
	blockHeight := lhprimitives.BlockHeight(genesisBlock.TransactionsBlock.Header.BlockHeight() + 1)

	waitTimeForMinimalBlockTransactions := config.ConsensusContextMinimalBlockTime()
	consensusRoundTime := waitTimeForMinimalBlockTransactions * 10
	provider := NewBlockProvider(logger, blockStorage, consensusContext, waitTimeForMinimalBlockTransactions, config.NodePublicKey(), config.NodePrivateKey())

	// TODO Configure to be 5 times the minimum wait for transactions (consensus context)

	electionTrigger := leanhelix.NewTimerBasedElectionTrigger(consensusRoundTime)

	s := &service{
		comm:         comm,
		blockStorage: blockStorage,
		logger:       logger,
		config:       config,
		metrics:      newMetrics(metricFactory, config.LeanHelixConsensusRoundTimeoutInterval()),
		leanHelix:    nil,
	}

	leanHelixConfig := &leanhelix.Config{
		NetworkCommunication: comm,
		BlockUtils:           provider,
		KeyManager:           mgr,
		ElectionTrigger:      electionTrigger,
		Logger:               NewLoggerWrapper(parentLogger, true),
	}

	leanHelix := leanhelix.NewLeanHelix(leanHelixConfig)

	s.leanHelix = leanHelix

	gossip.RegisterLeanHelixHandler(s)
	blockStorage.RegisterConsensusBlocksHandler(s)

	s.leanHelix.RegisterOnCommitted(func(block leanhelix.Block) {
		parentLogger.Info("YEYYYY CONSENSUS!!!!", log.Stringable("block-height", block.Height()))
		blockPairWrapper := block.(*BlockPairWrapper)
		blockPair := blockPairWrapper.blockPair
		s.saveToBlockStorage(ctx, blockPair)
	})

	if config.ActiveConsensusAlgo() == consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX {
		parentLogger.Info("Lean Helix go routine starts", log.BlockHeight(primitives.BlockHeight(blockHeight)))

		go s.leanHelix.Run(ctx) // TODO Get the block height from someplace

		s.leanHelix.AcknowledgeBlockConsensus(ToBlockPairWrapper(genesisBlock))

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

	prevBlock := ToBlockPairWrapper(prevCommittedBlockPair)

	s.leanHelix.AcknowledgeBlockConsensus(prevBlock)

	return nil, nil
}

func (s *service) HandleLeanHelixMessage(ctx context.Context, input *gossiptopics.LeanHelixInput) (*gossiptopics.EmptyOutput, error) {
	messageType := input.Message.MessageType
	s.logger.Info("leanhelix comm.HandleLeanHelixMessage()", log.Stringable("message-type", messageType))
	message := leanhelix.CreateConsensusRawMessage(
		leanhelix.MessageType(messageType),
		input.Message.Content,
		&BlockPairWrapper{
			blockPair: input.Message.BlockPair,
		},
	)
	s.leanHelix.GossipMessageReceived(ctx, message)
	return nil, nil
}

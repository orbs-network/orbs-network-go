package leanhelixconsensus

import (
	"context"
	"github.com/orbs-network/lean-helix-go"
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
	blockStorage  services.BlockStorage
	membership    *membership
	comm          *communication
	blockProvider *blockProvider
	logger        log.BasicLogger
	config        Config
	metrics       *metrics
	leanHelix     *leanhelix.LeanHelix
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

	provider := NewBlockProvider(logger, blockStorage, consensusContext, config.NodeAddress(), config.NodePrivateKey())

	// Configure to be ~5 times the minimum wait for transactions (consensus context)
	electionTrigger := leanhelix.NewTimerBasedElectionTrigger(config.LeanHelixConsensusRoundTimeoutInterval())

	s := &service{
		comm:          comm,
		blockStorage:  blockStorage,
		logger:        logger,
		config:        config,
		blockProvider: provider,
		metrics:       newMetrics(metricFactory, config.LeanHelixConsensusRoundTimeoutInterval()),
		leanHelix:     nil,
	}

	leanHelixConfig := &leanhelix.Config{
		Communication:   comm,
		Membership:      membership,
		BlockUtils:      provider,
		KeyManager:      mgr,
		ElectionTrigger: electionTrigger,
		Logger:          NewLoggerWrapper(parentLogger, true),
	}

	logger.Info("NewLeanHelixConsensusAlgo() run NewLeanHelix()")
	s.leanHelix = leanhelix.NewLeanHelix(leanHelixConfig, s.onCommit)

	// Note: LeanHelix could be used as handler to validateBlocks without actively running consensus rounds
	parentLogger.Info("LeanHelix go routine starts")
	supervised.GoForever(ctx, logger, func() {
		s.leanHelix.Run(ctx)
	})

	gossip.RegisterLeanHelixHandler(s)
	blockStorage.RegisterConsensusBlocksHandler(s)

	logger.Info("NewLeanHelixConsensusAlgo() active algo", log.Stringable("active-consensus-algo", config.ActiveConsensusAlgo()))

	return s
}

// TODO Go over this carefully!!
func (s *service) onCommit(ctx context.Context, block leanhelix.Block, blockProof []byte) {
	// log
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	logger.Info("YEYYYY CONSENSUS!!!! will save to block storage", log.Stringable("block-height", block.Height()))
	// convert block with proof to comply to blockstorage
	blockPairWrapper := block.(*BlockPairWrapper)
	blockPair := blockPairWrapper.blockPair
	// set blockProof
	// generate and set tx block proof
	blockPair.TransactionsBlock.BlockProof = (&protocol.TransactionsBlockProofBuilder{
		Type:             protocol.TRANSACTIONS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		ResultsBlockHash: digest.CalcResultsBlockHash(blockPair.ResultsBlock),
		LeanHelix:        blockProof,
	}).Build()
	// generate rx block proof
	blockPair.ResultsBlock.BlockProof = (&protocol.ResultsBlockProofBuilder{
		Type:                  protocol.RESULTS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		TransactionsBlockHash: digest.CalcTransactionsBlockHash(blockPair.TransactionsBlock),
		LeanHelix:             blockProof,
	}).Build()

	err := s.saveToBlockStorage(ctx, blockPair)
	if err != nil {
		logger.Info("onCommit - saving block to storage error: ", log.BlockHeight(blockPair.TransactionsBlock.Header.BlockHeight()))
	}

}

func (s *service) saveToBlockStorage(ctx context.Context, blockPair *protocol.BlockPairContainer) error {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	if blockPair.TransactionsBlock.Header.BlockHeight() == 0 {
		return errors.Errorf("saveToBlockStorage with block height 0 - genesis is not supported")
	}
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

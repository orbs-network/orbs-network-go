package leanhelixconsensus

import (
	"context"
	"github.com/orbs-network/lean-helix-go"
	"github.com/orbs-network/lean-helix-go/services/electiontrigger"
	lh "github.com/orbs-network/lean-helix-go/services/interfaces"
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
	com           *communication
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
	LeanHelixShowDebug() bool
	ActiveConsensusAlgo() consensus.ConsensusAlgoType
	VirtualChainId() primitives.VirtualChainId
	NetworkType() protocol.SignerNetworkType
}

func newMetrics(m metric.Factory, consensusTimeout time.Duration) *metrics {
	return &metrics{
		consensusRoundTickTime:     m.NewLatency("ConsensusAlgo.LeanHelix.RoundTickTime", consensusTimeout),
		failedConsensusTicksRate:   m.NewRate("ConsensusAlgo.LeanHelix.FailedTicksPerSecond"),
		timedOutConsensusTicksRate: m.NewRate("ConsensusAlgo.LeanHelix.TimedOutTicksPerSecond"),
	}
}

func NewLeanHelixConsensusAlgo(
	parentContext context.Context,
	gossip gossiptopics.LeanHelix,
	blockStorage services.BlockStorage,
	consensusContext services.ConsensusContext,
	parentLogger log.BasicLogger,
	config Config,
	metricFactory metric.Factory,

) services.ConsensusAlgoLeanHelix {

	ctx := trace.NewContext(parentContext, "LeanHelix.Run")
	logger := parentLogger.WithTags(LogTag, trace.LogFieldFrom(ctx))

	logger.Info("NewLeanHelixConsensusAlgo() start", log.String("Node-address", config.NodeAddress().String()))
	com := NewCommunication(logger, gossip)
	committeeSize := uint32(len(config.FederationNodes(0)))
	membership := NewMembership(logger, config.NodeAddress(), consensusContext, committeeSize)
	mgr := NewKeyManager(logger, config.NodePrivateKey())

	provider := NewBlockProvider(logger, blockStorage, consensusContext)

	// Configure to be ~5 times the minimum wait for transactions (consensus context)
	electionTrigger := electiontrigger.NewTimerBasedElectionTrigger(config.LeanHelixConsensusRoundTimeoutInterval())
	logger.Info("Election trigger set", log.String("election-trigger-timeout", config.LeanHelixConsensusRoundTimeoutInterval().String()))
	instanceId := CalcInstanceId(config.NetworkType(), config.VirtualChainId())

	s := &service{
		com:           com,
		blockStorage:  blockStorage,
		logger:        logger,
		config:        config,
		blockProvider: provider,
		metrics:       newMetrics(metricFactory, config.LeanHelixConsensusRoundTimeoutInterval()),
		leanHelix:     nil,
	}

	leanHelixConfig := &lh.Config{
		InstanceId:      instanceId,
		Communication:   com,
		Membership:      membership,
		BlockUtils:      provider,
		KeyManager:      mgr,
		ElectionTrigger: electionTrigger,
		Logger:          NewLoggerWrapper(parentLogger, config.LeanHelixShowDebug()),
	}

	logger.Info("NewLeanHelixConsensusAlgo() run NewLeanHelix()")
	s.leanHelix = leanhelix.NewLeanHelix(leanHelixConfig, s.onCommit)

	if config.ActiveConsensusAlgo() == consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX {
		supervised.GoForever(ctx, logger, func() {
			logger.Info("LeanHelix is active consensus algo: go routine starts")
			s.leanHelix.Run(ctx)
		})
		gossip.RegisterLeanHelixHandler(s)
	} else {
		logger.Info("LeanHelix is not the active consensus algo so not running its go routine, only registering for block validation")
	}
	// LeanHelix can be used as handler to validateBlocks without actively running consensus rounds
	blockStorage.RegisterConsensusBlocksHandler(s)

	logger.Info("NewLeanHelixConsensusAlgo() active algo", log.Stringable("active-consensus-algo", config.ActiveConsensusAlgo()))

	return s
}

func (s *service) HandleBlockConsensus(ctx context.Context, input *handlers.HandleBlockConsensusInput) (*handlers.HandleBlockConsensusOutput, error) {

	blockType := input.BlockType
	blockPair := input.BlockPair
	prevCommittedBlockPair := input.PrevCommittedBlockPair
	var lhBlockProof []byte
	var lhBlock lh.Block

	if blockType != protocol.BLOCK_TYPE_BLOCK_PAIR {
		return nil, errors.Errorf("handler received unsupported block type %s", blockType)
	}

	// validate the lhBlock consensus (lhBlock and proof)
	if shouldValidate(input.Mode) {
		err := s.validateBlockConsensus(ctx, blockPair, prevCommittedBlockPair)
		if err != nil {
			return nil, err
		}
	}

	// update the lhBlock consensus - with lhBlock and proof - (might be nil -> genesisBlock)
	if shouldUpdate(input.Mode) {
		// if LeanHelix is not the active consensus do not update
		//  Note: genesis case (nil) is special no lhBlockProof type - registered handlers cannot distinguish - should be handled here
		if s.config.ActiveConsensusAlgo() != consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX {
			s.logger.Info("LeanHelix is not the active consensus algo, not calling UpdateState()")
			// TODO: maybe add output in this case? (change protos - HandleBlockConsensusOutput?)
			return nil, nil
		}

		if shouldCreateGenesisBlock(blockPair) {
			lhBlock, lhBlockProof = s.blockProvider.GenerateGenesisBlockProposal(ctx)
			s.logger.Info("HandleBlockConsensus Update LeanHelix with GenesisBlock", log.Stringable("mode", input.Mode), log.Stringable("blockPair", blockPair))
		} else { // we should have a lhBlock proof
			s.logger.Info("HandleBlockConsensus Update LeanHelix with block", log.Stringable("mode", input.Mode), log.BlockHeight(blockPair.TransactionsBlock.Header.BlockHeight()))
			lhBlockProof = blockPair.TransactionsBlock.BlockProof.Raw()
			lhBlock = ToLeanHelixBlock(blockPair)

		}

		s.leanHelix.UpdateState(ctx, lhBlock, lhBlockProof)
		// TODO: Should we notify error?
	}

	return nil, nil
}

func (s *service) HandleLeanHelixMessage(ctx context.Context, input *gossiptopics.LeanHelixInput) (*gossiptopics.EmptyOutput, error) {
	consensusRawMessage := &lh.ConsensusRawMessage{
		Content: input.Message.Content,
		Block:   ToLeanHelixBlock(input.Message.BlockPair),
	}
	s.leanHelix.HandleConsensusMessage(ctx, consensusRawMessage)
	return nil, nil
}

func (s *service) onCommit(ctx context.Context, block lh.Block, blockProof []byte) {
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

func shouldCreateGenesisBlock(blockPair *protocol.BlockPairContainer) bool {
	return blockPair == nil
}

func shouldValidate(mode handlers.HandleBlockConsensusMode) bool {
	return mode == handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE || mode == handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_ONLY
}

func shouldUpdate(mode handlers.HandleBlockConsensusMode) bool {
	return mode == handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE || mode == handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY
}

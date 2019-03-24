// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package leanhelixconsensus

import (
	"context"
	"github.com/orbs-network/lean-helix-go"
	lhmetrics "github.com/orbs-network/lean-helix-go/instrumentation/metrics"
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
	blockStorage     services.BlockStorage
	membership       *membership
	com              *communication
	blockProvider    *blockProvider
	logger           log.BasicLogger
	config           config.LeanHelixConsensusConfig
	metrics          *metrics
	leanHelix        *leanhelix.LeanHelix
	lastCommitTime   time.Time
	lastElectionTime time.Time
}

type metrics struct {
	timeSinceLastCommitMillis   *metric.Histogram
	timeSinceLastElectionMillis *metric.Histogram
	currentLeaderMemberId       *metric.Text
	currentElectionCount        *metric.Gauge
	lastCommittedTime           *metric.Gauge
}

func newMetrics(m metric.Factory) *metrics {
	return &metrics{
		timeSinceLastCommitMillis:   m.NewLatency("ConsensusAlgo.LeanHelix.TimeSinceLastCommit.Millis", 30*time.Minute),
		timeSinceLastElectionMillis: m.NewLatency("ConsensusAlgo.LeanHelix.TimeSinceLastElection.Millis", 30*time.Minute),
		currentElectionCount:        m.NewGauge("ConsensusAlgo.LeanHelix.CurrentElection.Number"),
		currentLeaderMemberId:       m.NewText("ConsensusAlgo.LeanHelix.CurrentLeaderMemberId.Number"),
		lastCommittedTime:           m.NewGauge("ConsensusAlgo.LeanHelix.LastCommitted.TimeNano"),
	}
}

func NewLeanHelixConsensusAlgo(
	parentContext context.Context,
	gossip gossiptopics.LeanHelix,
	blockStorage services.BlockStorage,
	consensusContext services.ConsensusContext,
	parentLogger log.BasicLogger,
	config config.LeanHelixConsensusConfig,
	metricFactory metric.Factory,

) services.ConsensusAlgoLeanHelix {
	ctx := trace.NewContext(parentContext, "LeanHelix.Run")

	logger := parentLogger.WithTags(LogTag, trace.LogFieldFrom(ctx))

	logger.Info("NewLeanHelixConsensusAlgo() start", log.String("node-address", config.NodeAddress().String()))
	com := NewCommunication(logger, gossip)
	membership := NewMembership(logger, config.NodeAddress(), consensusContext, config.LeanHelixConsensusMaximumCommitteeSize())
	mgr := NewKeyManager(logger, config.NodePrivateKey())

	provider := NewBlockProvider(logger, blockStorage, consensusContext)

	instanceId := CalcInstanceId(config.NetworkType(), config.VirtualChainId())

	s := &service{
		com:           com,
		blockStorage:  blockStorage,
		logger:        logger,
		config:        config,
		blockProvider: provider,
		metrics:       newMetrics(metricFactory),
		leanHelix:     nil,
	}

	// TODO https://github.com/orbs-network/orbs-network-go/issues/786 Implement election trigger here, run its goroutine under "supervised"
	electionTrigger := NewExponentialBackoffElectionTrigger(logger, config.LeanHelixConsensusRoundTimeoutInterval(), s.onElection) // Configure to be ~5 times the minimum wait for transactions (consensus context)
	logger.Info("Election trigger set the first time", log.String("election-trigger-timeout", config.LeanHelixConsensusRoundTimeoutInterval().String()))

	leanHelixConfig := &lh.Config{
		InstanceId:      instanceId,
		Communication:   com,
		Membership:      membership,
		BlockUtils:      provider,
		KeyManager:      mgr,
		ElectionTrigger: electionTrigger,
		Logger:          NewLoggerWrapper(parentLogger, config.LeanHelixShowDebug()),
	}

	logger.Info("NewLeanHelixConsensusAlgo() instantiating NewLeanHelix() (not starting its goroutine yet)")
	s.leanHelix = leanhelix.NewLeanHelix(leanHelixConfig, s.onCommit)

	if config.ActiveConsensusAlgo() == consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX {
		supervised.GoForever(ctx, logger, func() {
			parentLogger.Info("NewLeanHelixConsensusAlgo() LeanHelix is active consensus algo: starting its goroutine")
			s.leanHelix.Run(ctx)
		})
		gossip.RegisterLeanHelixHandler(s)
	} else {
		parentLogger.Info("NewLeanHelixConsensusAlgo() LeanHelix is not the active consensus algo so not starting its goroutine, only registering for block validation")
	}
	// Do this even if not active consensus algo - LeanHelix can still be a handler to validateBlocks without actively running consensus rounds
	blockStorage.RegisterConsensusBlocksHandler(s)

	return s
}

func (s *service) HandleBlockConsensus(ctx context.Context, input *handlers.HandleBlockConsensusInput) (*handlers.HandleBlockConsensusOutput, error) {

	blockType := input.BlockType
	blockPair := input.BlockPair
	prevBlockPair := input.PrevCommittedBlockPair
	var lhBlockProof []byte
	var lhBlock lh.Block

	if blockType != protocol.BLOCK_TYPE_BLOCK_PAIR {
		return nil, errors.Errorf("HandleBlockConsensus(): LeanHelix: received unsupported block type %s", blockType)
	}

	// validate the lhBlock consensus (lhBlock and proof)
	if shouldValidateBlockConsensusWithLeanHelix(input.Mode) {
		err := s.validateBlockConsensus(ctx, blockPair, prevBlockPair)
		if err != nil {
			s.logger.Info("HandleBlockConsensus(): Failed validating block consensus with LeanHelix", log.Error(err))
			return nil, err
		}
	}

	if shouldUpdateStateInLeanHelix(input.Mode) {
		if s.config.ActiveConsensusAlgo() != consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX {
			s.logger.Info("HandleBlockConsensus(): LeanHelix is not the active consensus algo, not calling UpdateState()")
			return nil, nil
		}

		if shouldCreateGenesisBlock(blockPair) {
			lhBlock, lhBlockProof = s.blockProvider.GenerateGenesisBlockProposal(ctx)
			s.logger.Info("HandleBlockConsensus(): Calling UpdateState in LeanHelix with GenesisBlock", log.Stringable("mode", input.Mode))
		} else { // we should have a lhBlock proof
			s.logger.Info("HandleBlockConsensus(): Calling UpdateState in LeanHelix with block", log.Stringable("mode", input.Mode), log.BlockHeight(blockPair.TransactionsBlock.Header.BlockHeight()))
			var err error
			lhBlockProof, err = ExtractBlockProof(blockPair)
			if err != nil {
				return nil, err
			}
			lhBlock = ToLeanHelixBlock(blockPair)

		}

		// do not add a "go" command here (so this step becomes async tell and doesn't block the block sync) because we want to control the sync rate
		s.leanHelix.UpdateState(ctx, lhBlock, lhBlockProof)
	}

	return nil, nil
}

func ExtractBlockProof(blockPair *protocol.BlockPairContainer) (primitives.LeanHelixBlockProof, error) {
	if blockPair == nil || blockPair.TransactionsBlock == nil || blockPair.TransactionsBlock.BlockProof == nil {
		return nil, errors.New("blockPair or TransactionsBlock or BlockProof is nil")
	}
	if !blockPair.TransactionsBlock.BlockProof.IsTypeLeanHelix() {
		return nil, errors.New("BlockProof is not of type LeanHelix")
	}
	return blockPair.TransactionsBlock.BlockProof.LeanHelix(), nil
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
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	logger.Info("YEYYYY CONSENSUS!!!! will save to block storage", log.BlockHeight(primitives.BlockHeight(block.Height())))
	blockPairWrapper := block.(*BlockPairWrapper)
	blockPair := blockPairWrapper.blockPair

	blockPair.TransactionsBlock.BlockProof = CreateTransactionBlockProof(blockPair, blockProof)

	blockPair.ResultsBlock.BlockProof = CreateResultsBlockProof(blockPair, blockProof)

	err := s.saveToBlockStorage(ctx, blockPair)
	if err != nil {
		logger.Info("onCommit - saving block to storage error: ", log.BlockHeight(blockPair.TransactionsBlock.Header.BlockHeight()))
	}
	now := time.Now()
	s.metrics.lastCommittedTime.Update(now.UnixNano())
	s.metrics.timeSinceLastCommitMillis.RecordSince(s.lastCommitTime)
	s.lastCommitTime = now
}

func CreateResultsBlockProof(blockPair *protocol.BlockPairContainer, blockProof []byte) *protocol.ResultsBlockProof {
	return (&protocol.ResultsBlockProofBuilder{
		Type:                  protocol.RESULTS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		TransactionsBlockHash: digest.CalcTransactionsBlockHash(blockPair.TransactionsBlock),
		LeanHelix:             blockProof,
	}).Build()
}

func CreateTransactionBlockProof(blockPair *protocol.BlockPairContainer, blockProof []byte) *protocol.TransactionsBlockProof {
	return (&protocol.TransactionsBlockProofBuilder{
		Type:             protocol.TRANSACTIONS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		ResultsBlockHash: digest.CalcResultsBlockHash(blockPair.ResultsBlock),
		LeanHelix:        blockProof,
	}).Build()
}

func (s *service) onElection(m lhmetrics.ElectionMetrics) {
	memberIdStr := m.CurrentLeaderMemberId().String()[:6]
	s.metrics.currentLeaderMemberId.Update(string(memberIdStr))
	s.metrics.currentElectionCount.Update(int64(m.CurrentView()))
	now := time.Now()
	s.metrics.timeSinceLastElectionMillis.RecordSince(s.lastElectionTime)
	s.lastElectionTime = now
	s.logger.Info("onElection()", log.String("lh-leader-member-id", memberIdStr), log.Int64("lh-view", int64(m.CurrentView())))
}

func (s *service) saveToBlockStorage(ctx context.Context, blockPair *protocol.BlockPairContainer) error {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	if blockPair.TransactionsBlock.Header.BlockHeight() == 0 {
		return errors.Errorf("saveToBlockStorage with block height 0 - genesis is not supported")
	}
	hash := digest.CalcBlockHash(blockPair.TransactionsBlock, blockPair.ResultsBlock)
	logger.Info("saving block to storage", log.Stringable("block-hash", hash), log.BlockHeight(blockPair.TransactionsBlock.Header.BlockHeight()))
	_, err := s.blockStorage.CommitBlock(ctx, &services.CommitBlockInput{
		BlockPair: blockPair,
	})
	return err
}

func shouldCreateGenesisBlock(blockPair *protocol.BlockPairContainer) bool {
	return blockPair == nil
}

func shouldValidateBlockConsensusWithLeanHelix(mode handlers.HandleBlockConsensusMode) bool {
	return mode == handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE || mode == handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_ONLY
}

func shouldUpdateStateInLeanHelix(mode handlers.HandleBlockConsensusMode) bool {
	return mode == handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE || mode == handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY
}

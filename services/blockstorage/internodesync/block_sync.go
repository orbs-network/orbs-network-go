// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package internodesync

import (
	"context"
	"fmt"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/scribe/log"
	"time"
)

var LogTag = log.String("flow", "block-sync")

// this is coupled to gossip because the entire service is (block storage)
// nothing to gain right now in decoupling just the sync
type syncState interface {
	name() string
	String() string
	processState(ctx context.Context) syncState
}

type blockSyncConfig interface {
	NodeAddress() primitives.NodeAddress
	BlockSyncNumBlocksInBatch() uint32
	BlockSyncNoCommitInterval() time.Duration
	BlockSyncCollectResponseTimeout() time.Duration
	BlockSyncCollectChunksTimeout() time.Duration
	BlockSyncDescendingEnabled() bool
	CommitteeGracePeriod() time.Duration
}

type SyncState struct {
	TopBlock        *protocol.BlockPairContainer
	InOrderBlock    *protocol.BlockPairContainer
	LastSyncedBlock *protocol.BlockPairContainer
}

func (s *SyncState) GetSyncStateBlockHeights() (topHeight primitives.BlockHeight, inOrderHeight primitives.BlockHeight, lastSyncedHeight primitives.BlockHeight) {
	return getBlockHeight(s.TopBlock), getBlockHeight(s.InOrderBlock), getBlockHeight(s.LastSyncedBlock)
}

func (s *SyncState) String() string {
	if s == nil {
		return "<nil>"
	}
	topHeight, inOrderHeight, lastSyncedHeight := s.GetSyncStateBlockHeights()
	return fmt.Sprintf("{TopBlockHeight:%d,InOrderBlockHeight:%d,LastSyncedBlockHeight:%d}", uint64(topHeight), uint64(inOrderHeight), uint64(lastSyncedHeight))
}

func getBlockHeight(block *protocol.BlockPairContainer) primitives.BlockHeight {
	if block == nil {
		return 0
	}
	return block.TransactionsBlock.Header.BlockHeight()
}

type BlockSyncStorage interface {
	GetLastCommittedBlockHeight(ctx context.Context, input *services.GetLastCommittedBlockHeightInput) (*services.GetLastCommittedBlockHeightOutput, error)
	NodeSyncCommitBlock(ctx context.Context, input *services.CommitBlockInput) (*services.CommitBlockOutput, error)
	ValidateBlockForCommit(ctx context.Context, input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error)
	ValidateChainTip(ctx context.Context, input *services.ValidateChainTipInput) (*services.ValidateChainTipOutput, error)
	UpdateConsensusAlgosAboutLastCommittedBlockInLocalPersistence(ctx context.Context)
	GetBlock(height primitives.BlockHeight) (*protocol.BlockPairContainer, error)
	GetSyncState() SyncState
}

// state machine passes outside events into this channel type for consumption by the currently active state instance.
// within processState.processState() all states must read from the channel eagerly!
// keeping the channel clear for new incoming events and tossing out irrelevant messages.
type blockSyncConduit chan interface{}

func (c blockSyncConduit) drainAndCheckForShutdown(ctx context.Context) bool {
	for {
		select {
		case <-c: // nop
		case <-ctx.Done():
			return false // indicate a shutdown was signaled
		default:
			return true
		}
	}
}

type BlockSync struct {
	govnr.TreeSupervisor
	logger  log.Logger
	factory *stateFactory
	gossip  gossiptopics.BlockSync
	storage BlockSyncStorage
	conduit blockSyncConduit
	metrics *stateMachineMetrics
	config  blockSyncConfig
}

type stateMachineMetrics struct {
	statesTransitioned *metric.Gauge
}

func newStateMachineMetrics(factory metric.Factory) *stateMachineMetrics {
	return &stateMachineMetrics{
		statesTransitioned: factory.NewGauge("BlockSync.StateTransitions.Count"),
	}
}

func newBlockSyncWithFactory(ctx context.Context, config blockSyncConfig, factory *stateFactory, gossip gossiptopics.BlockSync, storage BlockSyncStorage, logger log.Logger, metricFactory metric.Factory) *BlockSync {
	metrics := newStateMachineMetrics(metricFactory)

	bs := &BlockSync{
		logger:  logger,
		factory: factory,
		gossip:  gossip,
		storage: storage,
		conduit: factory.conduit,
		metrics: metrics,
		config:  config,
	}

	logger.Info("Block sync init",
		log.Stringable("no-commit-timeout", bs.factory.config.BlockSyncNoCommitInterval()),
		log.Stringable("collect-responses-timeout", bs.factory.config.BlockSyncCollectResponseTimeout()),
		log.Stringable("collect-chunks-timeout", bs.factory.config.BlockSyncCollectChunksTimeout()),
		log.Uint32("batch-size", bs.factory.config.BlockSyncNumBlocksInBatch()),
		log.Stringable("blocks-order", bs.factory.getSyncBlocksOrder()),
		log.Stringable("committee-grace-period", bs.factory.config.CommitteeGracePeriod()))

	bs.Supervise(govnr.Forever(ctx, "Node sync state machine", logfields.GovnrErrorer(logger), func() {
		bs.syncLoop(ctx)
	}))

	return bs
}

func NewBlockSync(ctx context.Context, config blockSyncConfig, gossip gossiptopics.BlockSync, storage BlockSyncStorage, parentLogger log.Logger, metricFactory metric.Factory) *BlockSync {
	logger := parentLogger.WithTags(LogTag)

	conduit := make(blockSyncConduit)
	return newBlockSyncWithFactory(
		ctx,
		config,
		NewStateFactory(config, gossip, storage, conduit, logger, metricFactory),
		gossip,
		storage,
		logger,
		metricFactory,
	)
}

func (bs *BlockSync) syncLoop(parent context.Context) {
	for currentState := bs.factory.CreateCollectingAvailabilityResponseState(); currentState != nil; {
		ctx := trace.NewContext(parent, "BlockSync")
		bs.logger.Info("state transitioning", log.Stringable("current-state", currentState), trace.LogFieldFrom(ctx))

		currentState = currentState.processState(ctx)
		bs.metrics.statesTransitioned.Inc()
	}
}

func (bs *BlockSync) HandleBlockCommitted(ctx context.Context) {
	logger := bs.logger.WithTags(trace.LogFieldFrom(ctx))
	select {
	case bs.conduit <- idleResetMessage{}:
	case <-ctx.Done():
		logger.Info("terminated on handle block committed", log.Error(ctx.Err()))
	}
}

func (bs *BlockSync) HandleBlockAvailabilityResponse(ctx context.Context, input *gossiptopics.BlockAvailabilityResponseInput) (*gossiptopics.EmptyOutput, error) {
	logger := bs.logger.WithTags(trace.LogFieldFrom(ctx))

	select {
	case bs.conduit <- input.Message:
	case <-ctx.Done():
		logger.Info("terminated on writing new availability response",
			log.String("context-message", ctx.Err().Error()),
			log.Stringable("response-source", input.Message.Sender.SenderNodeAddress()))
	}
	return nil, nil
}

func (bs *BlockSync) HandleBlockSyncResponse(ctx context.Context, input *gossiptopics.BlockSyncResponseInput) (*gossiptopics.EmptyOutput, error) {
	logger := bs.logger.WithTags(trace.LogFieldFrom(ctx))

	select {
	case bs.conduit <- input.Message:
	case <-ctx.Done():
		logger.Info("terminated on writing new block chunk message",
			log.String("context-message", ctx.Err().Error()),
			log.Stringable("message-sender", input.Message.Sender.SenderNodeAddress()))
	}

	return nil, nil
}

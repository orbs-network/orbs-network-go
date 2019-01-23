package internodesync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
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
}

type BlockSyncStorage interface {
	GetLastCommittedBlockHeight(ctx context.Context, input *services.GetLastCommittedBlockHeightInput) (*services.GetLastCommittedBlockHeightOutput, error)
	NodeSyncCommitBlock(ctx context.Context, input *services.CommitBlockInput) (*services.CommitBlockOutput, error)
	ValidateBlockForCommit(ctx context.Context, input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error)
	UpdateConsensusAlgosAboutLatestCommittedBlock(ctx context.Context)
}

// the conduit connects between the states and the state machine (which is connected to the gossip handler)
// the data that the states receive, regardless of their instance, is waiting at these channels
type blockSyncConduit struct {
	done      chan struct{}
	idleReset chan struct{}
	responses chan *gossipmessages.BlockAvailabilityResponseMessage
	blocks    chan *gossipmessages.BlockSyncResponseMessage
}

type BlockSync struct {
	logger  log.BasicLogger
	factory *stateFactory
	gossip  gossiptopics.BlockSync
	storage BlockSyncStorage
	config  blockSyncConfig
	//currentState syncState
	conduit *blockSyncConduit

	metrics *stateMachineMetrics
}

type stateMachineMetrics struct {
	statesTransitioned *metric.Gauge
}

func newStateMachineMetrics(factory metric.Factory) *stateMachineMetrics {
	return &stateMachineMetrics{
		statesTransitioned: factory.NewGauge("BlockSync.StateTransitions"),
	}
}

func newBlockSyncWithFactory(ctx context.Context, factory *stateFactory, config blockSyncConfig, gossip gossiptopics.BlockSync, storage BlockSyncStorage, logger log.BasicLogger, metricFactory metric.Factory) *BlockSync {
	metrics := newStateMachineMetrics(metricFactory)

	bs := &BlockSync{
		logger:  logger,
		factory: factory,
		gossip:  gossip,
		storage: storage,
		config:  config,
		conduit: factory.conduit,
		metrics: metrics,
	}

	logger.Info("block sync init",
		log.Stringable("no-commit-timeout", bs.config.BlockSyncNoCommitInterval()),
		log.Stringable("collect-responses-timeout", bs.config.BlockSyncCollectResponseTimeout()),
		log.Stringable("collect-chunks-timeout", bs.config.BlockSyncCollectChunksTimeout()),
		log.Uint32("batch-size", bs.config.BlockSyncNumBlocksInBatch()))

	supervised.GoForever(ctx, logger, func() {
		bs.syncLoop(ctx)
	})

	return bs
}

func NewBlockSync(ctx context.Context, config blockSyncConfig, gossip gossiptopics.BlockSync, storage BlockSyncStorage, parentLogger log.BasicLogger, metricFactory metric.Factory) *BlockSync {
	logger := parentLogger.WithTags(LogTag)

	conduit := &blockSyncConduit{
		done:      make(chan struct{}),
		idleReset: make(chan struct{}),
		responses: make(chan *gossipmessages.BlockAvailabilityResponseMessage),
		blocks:    make(chan *gossipmessages.BlockSyncResponseMessage),
	}
	return newBlockSyncWithFactory(
		ctx,
		NewStateFactory(config, gossip, storage, conduit, logger, metricFactory),
		config,
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

	close(bs.conduit.done)
}

func (bs *BlockSync) IsTerminated() bool {
	select {
	case _, open := <-bs.conduit.done:
		return !open
	default:
		return false
	}
}

func (bs *BlockSync) HandleBlockCommitted(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, bs.config.BlockSyncNoCommitInterval()/2)
	defer cancel()

	select {
	case bs.conduit.idleReset <- struct{}{}:
	case <-ctx.Done():
	}
}

func (bs *BlockSync) HandleBlockAvailabilityResponse(ctx context.Context, input *gossiptopics.BlockAvailabilityResponseInput) (*gossiptopics.EmptyOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, bs.config.BlockSyncCollectResponseTimeout()/2)
	defer cancel()
	logger := bs.logger.WithTags(trace.LogFieldFrom(ctx))

	select {
	case bs.conduit.responses <- input.Message:
	case <-ctx.Done():
		logger.Info("terminated on writing new availability response",
			log.String("context-message", ctx.Err().Error()),
			log.Stringable("response-source", input.Message.Sender.SenderNodeAddress()))
	}
	return nil, nil
}

func (bs *BlockSync) HandleBlockSyncResponse(ctx context.Context, input *gossiptopics.BlockSyncResponseInput) (*gossiptopics.EmptyOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, bs.config.BlockSyncCollectChunksTimeout()/2)
	defer cancel()
	logger := bs.logger.WithTags(trace.LogFieldFrom(ctx))

	select {
	case bs.conduit.blocks <- input.Message:
	case <-ctx.Done():
		logger.Info("terminated on writing new block chunk message",
			log.String("context-message", ctx.Err().Error()),
			log.Stringable("message-sender", input.Message.Sender.SenderNodeAddress()))
	}

	return nil, nil
}

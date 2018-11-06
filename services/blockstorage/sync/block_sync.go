package sync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"time"
)

// this is coupled to gossip because the entire service is (block storage)
// nothing to gain right now in decoupling just the sync
type syncState interface {
	name() string
	String() string
	processState(ctx context.Context) syncState
	blockCommitted(ctx context.Context)
	gotAvailabilityResponse(ctx context.Context, message *gossipmessages.BlockAvailabilityResponseMessage)
	gotBlocks(ctx context.Context, message *gossipmessages.BlockSyncResponseMessage)
}

type blockSyncConfig interface {
	NodePublicKey() primitives.Ed25519PublicKey
	BlockSyncBatchSize() uint32
	BlockSyncNoCommitInterval() time.Duration
	BlockSyncCollectResponseTimeout() time.Duration
	BlockSyncCollectChunksTimeout() time.Duration
}

type BlockSyncStorage interface {
	LastCommittedBlockHeight() primitives.BlockHeight
	CommitBlock(ctx context.Context, input *services.CommitBlockInput) (*services.CommitBlockOutput, error)
	ValidateBlockForCommit(ctx context.Context, input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error)
	UpdateConsensusAlgosAboutLatestCommittedBlock(ctx context.Context)
}

// the conduit connects between the states and the state machine (which is connected to the gossip handler)
// the data that the states receive, regardless of their instance, is waiting at these channels
type blockSyncConduit struct {
	idleReset chan struct{}
	responses chan *gossipmessages.BlockAvailabilityResponseMessage
	blocks    chan *gossipmessages.BlockSyncResponseMessage
}

type BlockSync struct {
	logger       log.BasicLogger
	sf           *stateFactory
	gossip       gossiptopics.BlockSync
	storage      BlockSyncStorage
	config       blockSyncConfig
	currentState syncState
	c            *blockSyncConduit

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

func NewBlockSync(ctx context.Context, config blockSyncConfig, gossip gossiptopics.BlockSync, storage BlockSyncStorage, logger log.BasicLogger, factory metric.Factory) *BlockSync {
	conduit := &blockSyncConduit{
		idleReset: make(chan struct{}),
		responses: make(chan *gossipmessages.BlockAvailabilityResponseMessage),
		blocks:    make(chan *gossipmessages.BlockSyncResponseMessage),
	}

	metrics := newStateMachineMetrics(factory)

	bs := &BlockSync{
		logger:  logger.WithTags(log.String("flow", "block-sync")),
		sf:      NewStateFactory(config, gossip, storage, conduit, logger, factory),
		gossip:  gossip,
		storage: storage,
		config:  config,
		c:       conduit,
		metrics: metrics,
	}

	bs.logger.Info("block sync init",
		log.Stringable("no-commit-timeout", bs.config.BlockSyncNoCommitInterval()),
		log.Stringable("collect-responses-timeout", bs.config.BlockSyncCollectResponseTimeout()),
		log.Stringable("collect-chunks-timeout", bs.config.BlockSyncCollectChunksTimeout()),
		log.Uint32("batch-size", bs.config.BlockSyncBatchSize()))

	supervised.GoForever(ctx, logger, func() {
		bs.syncLoop(ctx)
	})

	return bs
}

func (bs *BlockSync) syncLoop(ctx context.Context) {
	for bs.currentState = bs.sf.CreateCollectingAvailabilityResponseState(); bs.currentState != nil; {
		bs.logger.Info("state transitioning", log.Stringable("current-state", bs.currentState))

		bs.currentState = bs.currentState.processState(ctx)
		bs.metrics.statesTransitioned.Inc()
	}
}

func (bs *BlockSync) HandleBlockCommitted(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, bs.config.BlockSyncNoCommitInterval()/2)
	if bs.currentState != nil {
		bs.currentState.blockCommitted(ctx)
	}
	cancel()
}

func (bs *BlockSync) HandleBlockAvailabilityResponse(ctx context.Context, input *gossiptopics.BlockAvailabilityResponseInput) (*gossiptopics.EmptyOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, bs.config.BlockSyncCollectResponseTimeout()/2)
	if bs.currentState != nil {
		bs.currentState.gotAvailabilityResponse(ctx, input.Message)
	}
	cancel()
	return nil, nil
}

func (bs *BlockSync) HandleBlockSyncResponse(ctx context.Context, input *gossiptopics.BlockSyncResponseInput) (*gossiptopics.EmptyOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, bs.config.BlockSyncCollectChunksTimeout()/2)
	if bs.currentState != nil {
		bs.currentState.gotBlocks(ctx, input.Message)
	}
	cancel()
	return nil, nil
}

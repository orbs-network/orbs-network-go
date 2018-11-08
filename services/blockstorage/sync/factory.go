package sync

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"time"
)

type stateFactory struct {
	config                          blockSyncConfig
	gossip                          gossiptopics.BlockSync
	storage                         BlockSyncStorage
	conduit                         *blockSyncConduit
	createCollectTimeoutTimer       func() *synchronization.Timer
	createNoCommitTimeoutTimer      func() *synchronization.Timer
	createWaitForChunksTimeoutTimer func() *synchronization.Timer
	logger                          log.BasicLogger
	metrics                         *stateMetrics
}

func NewStateFactory(
	config blockSyncConfig,
	gossip gossiptopics.BlockSync,
	storage BlockSyncStorage,
	conduit *blockSyncConduit,
	logger log.BasicLogger,
	factory metric.Factory,
) *stateFactory {
	return NewStateFactoryWithTimers(
		config,
		gossip,
		storage,
		conduit,
		nil,
		nil,
		nil,
		logger,
		factory)
}

func NewStateFactoryWithTimers(
	config blockSyncConfig,
	gossip gossiptopics.BlockSync,
	storage BlockSyncStorage,
	conduit *blockSyncConduit,
	createCollectTimeoutTimer func() *synchronization.Timer,
	createNoCommitTimeoutTimer func() *synchronization.Timer,
	createWaitForChunksTimeoutTimer func() *synchronization.Timer,
	logger log.BasicLogger,
	factory metric.Factory,
) *stateFactory {

	f := &stateFactory{
		config:  config,
		gossip:  gossip,
		storage: storage,
		conduit: conduit,
		logger:  logger,
		metrics: newStateMetrics(factory),
	}

	if createCollectTimeoutTimer == nil {
		f.createCollectTimeoutTimer = f.defaultCreateCollectTimeoutTimer
	} else {
		f.createCollectTimeoutTimer = createCollectTimeoutTimer
	}

	if createNoCommitTimeoutTimer == nil {
		f.createNoCommitTimeoutTimer = f.defaultCreateNoCommitTimeoutTimer
	} else {
		f.createNoCommitTimeoutTimer = createNoCommitTimeoutTimer
	}

	if createWaitForChunksTimeoutTimer == nil {
		f.createWaitForChunksTimeoutTimer = f.defaultCreateWaitForChunksTimeoutTimer
	} else {
		f.createWaitForChunksTimeoutTimer = createWaitForChunksTimeoutTimer
	}

	return f
}

func (f *stateFactory) defaultCreateCollectTimeoutTimer() *synchronization.Timer {
	return synchronization.NewTimer(f.config.BlockSyncCollectResponseTimeout())
}

func (f *stateFactory) defaultCreateNoCommitTimeoutTimer() *synchronization.Timer {
	return synchronization.NewTimer(f.config.BlockSyncNoCommitInterval())
}

func (f *stateFactory) defaultCreateWaitForChunksTimeoutTimer() *synchronization.Timer {
	return synchronization.NewTimer(f.config.BlockSyncCollectChunksTimeout())
}

func (f *stateFactory) CreateIdleState() syncState {
	return &idleState{
		factory:                    f,
		createNoCommitTimeoutTimer: f.createNoCommitTimeoutTimer,
		logger:  f.logger,
		conduit: f.conduit,
		metrics: f.metrics.idleStateMetrics,
	}
}

func (f *stateFactory) CreateCollectingAvailabilityResponseState() syncState {
	return &collectingAvailabilityResponsesState{
		factory:                   f,
		gossipClient:              newBlockSyncGossipClient(f.gossip, f.storage, f.logger, f.config.BlockSyncBatchSize, f.config.NodePublicKey),
		createCollectTimeoutTimer: f.createCollectTimeoutTimer,
		logger:  f.logger,
		conduit: f.conduit,
		metrics: f.metrics.collectingStateMetrics,
	}
}

func (f *stateFactory) CreateFinishedCARState(responses []*gossipmessages.BlockAvailabilityResponseMessage) syncState {
	return &finishedCARState{
		responses: responses,
		logger:    f.logger,
		factory:   f,
		metrics:   f.metrics.finishedCollectingStateMetrics,
	}
}

func (f *stateFactory) CreateWaitingForChunksState(sourceKey primitives.Ed25519PublicKey) syncState {
	return &waitingForChunksState{
		sourceKey:                       sourceKey,
		factory:                         f,
		gossipClient:                    newBlockSyncGossipClient(f.gossip, f.storage, f.logger, f.config.BlockSyncBatchSize, f.config.NodePublicKey),
		createWaitForChunksTimeoutTimer: f.createWaitForChunksTimeoutTimer,
		logger:  f.logger,
		abort:   make(chan struct{}),
		conduit: f.conduit,
		metrics: f.metrics.waitingStateMetrics,
	}
}

func (f *stateFactory) CreateProcessingBlocksState(message *gossipmessages.BlockSyncResponseMessage) syncState {
	return &processingBlocksState{
		blocks:  message,
		factory: f,
		logger:  f.logger,
		storage: f.storage,
		metrics: f.metrics.processingStateMetrics,
	}
}

type stateMetrics struct {
	idleStateMetrics
	collectingStateMetrics
	finishedCollectingStateMetrics
	waitingStateMetrics
	processingStateMetrics
}

type idleStateMetrics struct {
	stateLatency *metric.Histogram
	timesReset   *metric.Gauge
	timesExpired *metric.Gauge
}

type collectingStateMetrics struct {
	stateLatency    *metric.Histogram
	timesSuccessful *metric.Gauge
}

type finishedCollectingStateMetrics struct {
	stateLatency       *metric.Histogram
	timesNoResponses   *metric.Gauge
	timesWithResponses *metric.Gauge
}

type waitingStateMetrics struct {
	stateLatency    *metric.Histogram
	timesTimeout    *metric.Gauge
	timesSuccessful *metric.Gauge
	timesByzantine  *metric.Gauge
}

type processingStateMetrics struct {
	stateLatency           *metric.Histogram
	blocksRate             *metric.Rate
	committedBlocks        *metric.Gauge
	failedCommitBlocks     *metric.Gauge
	failedValidationBlocks *metric.Gauge
}

func newStateMetrics(factory metric.Factory) *stateMetrics {
	return &stateMetrics{
		idleStateMetrics: idleStateMetrics{
			stateLatency: factory.NewLatency("BlockSync.Idle.StateLatency", 24*30*time.Hour),
			timesReset:   factory.NewGauge("BlockSync.Idle.TimesReset"),
			timesExpired: factory.NewGauge("BlockSync.Idle.TimesExpired"),
		},
		collectingStateMetrics: collectingStateMetrics{
			stateLatency:    factory.NewLatency("BlockSync.Collecting.StateLatency", 24*30*time.Hour),
			timesSuccessful: factory.NewGauge("BlockSync.Collecting.SuccessCount"),
		},
		finishedCollectingStateMetrics: finishedCollectingStateMetrics{
			stateLatency:       factory.NewLatency("BlockSync.FinishedCollecting.StateLatency", 24*30*time.Hour),
			timesNoResponses:   factory.NewGauge("BlockSync.FinishedCollecting.NoResponsesCount"),
			timesWithResponses: factory.NewGauge("BlockSync.FinishedCollecting.WithResponsesCount"),
		},
		waitingStateMetrics: waitingStateMetrics{
			stateLatency:    factory.NewLatency("BlockSync.Waiting.StateLatency", 24*30*time.Hour),
			timesByzantine:  factory.NewGauge("BlockSync.Waiting.ByzantineResponseCount"),
			timesSuccessful: factory.NewGauge("BlockSync.Waiting.SuccessResponseCount"),
			timesTimeout:    factory.NewGauge("BlockSync.Waiting.TimeoutCount"),
		},
		processingStateMetrics: processingStateMetrics{
			stateLatency:           factory.NewLatency("BlockSync.Processing.StateLatency", 24*30*time.Hour),
			blocksRate:             factory.NewRate("BlockSync.Processing.BlocksRate"),
			committedBlocks:        factory.NewGauge("BlockSync.Processing.CommittedBlocks"),
			failedCommitBlocks:     factory.NewGauge("BlockSync.Processing.FailedToCommitBlocks"),
			failedValidationBlocks: factory.NewGauge("BlockSync.Processing.FailedToValidateBlocks"),
		},
	}
}

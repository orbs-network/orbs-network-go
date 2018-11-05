package sync

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"time"
)

type stateFactory struct {
	config  blockSyncConfig
	gossip  gossiptopics.BlockSync
	storage BlockSyncStorage
	c       *blockSyncConduit
	logger  log.BasicLogger
	metrics *stateMetrics
}

type stateMetrics struct {
	carStateLatency         *metric.Histogram
	finishedCarStateLatency *metric.Histogram
	idleStateMetrics
	waitingStateMetrics
	processingStateMetrics
}

type idleStateMetrics struct {
	stateLatency *metric.Histogram
	timesReset   *metric.Gauge
	timesExpired *metric.Gauge
}

type waitingStateMetrics struct {
	stateLatency    *metric.Histogram
	timesTimeout    *metric.Gauge
	timesSuccessful *metric.Gauge
	timesByzanitine *metric.Gauge
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
		carStateLatency:         factory.NewLatency("BlockSync.State.CollectingStateLatency", 24*30*time.Hour),
		finishedCarStateLatency: factory.NewLatency("BlockSync.State.FinishedCollectingStateLatency", 24*30*time.Hour),
		idleStateMetrics: idleStateMetrics{
			stateLatency: factory.NewLatency("BlockSync.Idle.StateLatency", 24*30*time.Hour),
			timesReset:   factory.NewGauge("BlockSync.Idle.TimesReset"),
			timesExpired: factory.NewGauge("BlockSync.Idle.TimesExpired"),
		},
		waitingStateMetrics: waitingStateMetrics{
			stateLatency:    factory.NewLatency("BlockSync.Waiting.StateLatency", 24*30*time.Hour),
			timesByzanitine: factory.NewGauge("BlockSync.Waiting.ByzantineResponseCount"),
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

func NewStateFactory(
	config blockSyncConfig,
	gossip gossiptopics.BlockSync,
	storage BlockSyncStorage,
	syncConduit *blockSyncConduit,
	logger log.BasicLogger,
	factory metric.Factory) *stateFactory {

	return &stateFactory{
		config:  config,
		gossip:  gossip,
		storage: storage,
		c:       syncConduit,
		logger:  logger,
		metrics: newStateMetrics(factory),
	}
}

func (f *stateFactory) CreateIdleState() syncState {
	return &idleState{
		sf:          f,
		idleTimeout: f.config.BlockSyncNoCommitInterval,
		logger:      f.logger,
		conduit:     f.c,
		m:           f.metrics.idleStateMetrics,
	}
}

func (f *stateFactory) CreateCollectingAvailabilityResponseState() syncState {
	return &collectingAvailabilityResponsesState{
		sf:             f,
		gossipClient:   newBlockSyncGossipClient(f.gossip, f.storage, f.logger, f.config.BlockSyncBatchSize, f.config.NodePublicKey),
		collectTimeout: f.config.BlockSyncCollectResponseTimeout,
		logger:         f.logger,
		conduit:        f.c,
		latency:        f.metrics.carStateLatency,
	}
}

func (f *stateFactory) CreateFinishedCARState(responses []*gossipmessages.BlockAvailabilityResponseMessage) syncState {
	return &finishedCARState{
		responses: responses,
		logger:    f.logger,
		sf:        f,
		latency:   f.metrics.finishedCarStateLatency,
	}
}

func (f *stateFactory) CreateWaitingForChunksState(sourceKey primitives.Ed25519PublicKey) syncState {
	return &waitingForChunksState{
		sourceKey:      sourceKey,
		sf:             f,
		gossipClient:   newBlockSyncGossipClient(f.gossip, f.storage, f.logger, f.config.BlockSyncBatchSize, f.config.NodePublicKey),
		collectTimeout: f.config.BlockSyncCollectChunksTimeout,
		logger:         f.logger,
		abort:          make(chan struct{}),
		conduit:        f.c,
		m:              f.metrics.waitingStateMetrics,
	}
}

func (f *stateFactory) CreateProcessingBlocksState(message *gossipmessages.BlockSyncResponseMessage) syncState {
	return &processingBlocksState{
		blocks:  message,
		sf:      f,
		logger:  f.logger,
		storage: f.storage,
		m:       f.metrics.processingStateMetrics,
	}
}

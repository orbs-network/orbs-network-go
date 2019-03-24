// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package internodesync

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
	conduit                         blockSyncConduit
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
	conduit blockSyncConduit,
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
	conduit blockSyncConduit,
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
		factory:     f,
		createTimer: f.createNoCommitTimeoutTimer,
		logger:      f.logger,
		conduit:     f.conduit,
		metrics:     f.metrics.idleStateMetrics,
	}
}

func (f *stateFactory) CreateCollectingAvailabilityResponseState() syncState {
	return &collectingAvailabilityResponsesState{
		factory:     f,
		client:      newBlockSyncGossipClient(f.gossip, f.storage, f.logger, f.config.BlockSyncNumBlocksInBatch, f.config.NodeAddress),
		createTimer: f.createCollectTimeoutTimer,
		logger:      f.logger,
		conduit:     f.conduit,
		metrics:     f.metrics.collectingStateMetrics,
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

func (f *stateFactory) CreateWaitingForChunksState(sourceNodeAddress primitives.NodeAddress) syncState {
	return &waitingForChunksState{
		sourceNodeAddress: sourceNodeAddress,
		factory:           f,
		client:            newBlockSyncGossipClient(f.gossip, f.storage, f.logger, f.config.BlockSyncNumBlocksInBatch, f.config.NodeAddress),
		createTimer:       f.createWaitForChunksTimeoutTimer,
		logger:            f.logger,
		conduit:           f.conduit,
		metrics:           f.metrics.waitingStateMetrics,
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
	timeSpentInState *metric.Histogram
	timesReset       *metric.Gauge
	timesExpired     *metric.Gauge
}

type collectingStateMetrics struct {
	timeSpentInState                         *metric.Histogram
	timesSucceededSendingAvailabilityRequest *metric.Gauge
	timesFailedSendingAvailabilityRequest    *metric.Gauge
}

type finishedCollectingStateMetrics struct {
	timeSpentInState               *metric.Histogram
	finishedWithNoResponsesCount   *metric.Gauge
	finishedWithSomeResponsesCount *metric.Gauge
}

type waitingStateMetrics struct {
	timeSpentInState *metric.Histogram
	timesTimeout     *metric.Gauge
	timesSuccessful  *metric.Gauge
	timesByzantine   *metric.Gauge
}

type processingStateMetrics struct {
	timeSpentInState       *metric.Histogram
	blocksRate             *metric.Rate
	committedBlocks        *metric.Gauge
	failedCommitBlocks     *metric.Gauge
	failedValidationBlocks *metric.Gauge
	lastCommittedTime      *metric.Gauge
}

func newStateMetrics(factory metric.Factory) *stateMetrics {
	return &stateMetrics{
		idleStateMetrics: idleStateMetrics{
			timeSpentInState: factory.NewLatency("BlockSync.IdleState.Duration.Millis", 24*30*time.Hour),
			timesReset:       factory.NewGauge("BlockSync.IdleState.ResetBackToIdle.Count"),
			timesExpired:     factory.NewGauge("BlockSync.IdleState.StartedBlockSync.Count"),
		},
		collectingStateMetrics: collectingStateMetrics{
			timeSpentInState:                         factory.NewLatency("BlockSync.CollectingAvailabilityResponsesState.Duration.Millis", 24*30*time.Hour),
			timesSucceededSendingAvailabilityRequest: factory.NewGauge("BlockSync.CollectingAvailabilityResponsesState.BroadcastSendSuccess.Count"),
			timesFailedSendingAvailabilityRequest:    factory.NewGauge("BlockSync.CollectingAvailabilityResponsesState.BroadcastSendFailure.Count"),
		},
		finishedCollectingStateMetrics: finishedCollectingStateMetrics{
			timeSpentInState:               factory.NewLatency("BlockSync.FinishedCollectingAvailabilityResponsesState.Duration.Millis", 24*30*time.Hour),
			finishedWithNoResponsesCount:   factory.NewGauge("BlockSync.FinishedCollectingAvailabilityResponsesState.FinishedWithNoResponses.Count"),
			finishedWithSomeResponsesCount: factory.NewGauge("BlockSync.FinishedCollectingAvailabilityResponsesState.FinishedWithSomeResponses.Count"),
		},
		waitingStateMetrics: waitingStateMetrics{
			timeSpentInState: factory.NewLatency("BlockSync.WaitingForBlocksState.Duration.Millis", 24*30*time.Hour),
			timesByzantine:   factory.NewGauge("BlockSync.WaitingForBlocksState.ReceivedBlocksFromByzantineSource.Count"),
			timesSuccessful:  factory.NewGauge("BlockSync.WaitingForBlocksState.ReceivedBlocksFromExpectedSource.Count"),
			timesTimeout:     factory.NewGauge("BlockSync.WaitingForBlocksState.TimedOutWithoutReceivingBlocks.Count"),
		},
		processingStateMetrics: processingStateMetrics{
			timeSpentInState:       factory.NewLatency("BlockSync.ProcessingBlocksState.Duration.Millis", 24*30*time.Hour),
			blocksRate:             factory.NewRate("BlockSync.ProcessingBlocksState.BlocksReceived.PerSecond"),
			committedBlocks:        factory.NewGauge("BlockSync.ProcessingBlocksState.CommittedBlocks.Count"),
			failedCommitBlocks:     factory.NewGauge("BlockSync.ProcessingBlocksState.FailedToCommitBlocks.Count"),
			failedValidationBlocks: factory.NewGauge("BlockSync.ProcessingBlocksState.FailedToValidateBlocks.Count"),
			lastCommittedTime:      factory.NewGauge("BlockSync.ProcessingBlocksState.LastCommitted.TimeNano"),
		},
	}
}

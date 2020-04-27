// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package internodesync

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/scribe/log"
	"time"
)

type headerStateFactory struct {
	config                          nodeSyncConfig
	gossip                          gossiptopics.HeaderSync
	storage                         BlockSyncStorage
	conduit                         headerSyncConduit
	createCollectTimeoutTimer       func() *synchronization.Timer
	createNoCommitTimeoutTimer      func() *synchronization.Timer
	createWaitForChunksTimeoutTimer func() *synchronization.Timer
	logger                          log.Logger
	metrics                         *stateMetrics
}

func NewHeaderStateFactory(
	config nodeSyncConfig,
	gossip gossiptopics.HeaderSync,
	storage BlockSyncStorage,
	conduit headerSyncConduit,
	logger log.Logger,
	factory metric.Factory,
) *headerStateFactory {
	return NewHeaderStateFactoryWithTimers(
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

func NewHeaderStateFactoryWithTimers(
	config nodeSyncConfig,
	gossip gossiptopics.HeaderSync,
	storage BlockSyncStorage,
	conduit headerSyncConduit,
	createCollectTimeoutTimer func() *synchronization.Timer,
	createNoCommitTimeoutTimer func() *synchronization.Timer,
	createWaitForChunksTimeoutTimer func() *synchronization.Timer,
	logger log.Logger,
	factory metric.Factory,
) *headerStateFactory {

	f := &headerStateFactory{
		config:  config,
		gossip:  gossip,
		storage: storage,
		conduit: conduit,
		logger:  logger,
		metrics: newHeadersStateMetrics(factory),
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

func (f *headerStateFactory) defaultCreateCollectTimeoutTimer() *synchronization.Timer {
	return synchronization.NewTimer(f.config.BlockSyncCollectResponseTimeout())
}

func (f *headerStateFactory) defaultCreateNoCommitTimeoutTimer() *synchronization.Timer {
	return synchronization.NewTimer(f.config.BlockSyncNoCommitInterval())
}

func (f *headerStateFactory) defaultCreateWaitForChunksTimeoutTimer() *synchronization.Timer {
	return synchronization.NewTimer(f.config.BlockSyncCollectChunksTimeout())
}

func (f *headerStateFactory) CreateIdleState() headerSyncState {
	return &idleHeadersState{
		factory:     f,
		createTimer: f.createNoCommitTimeoutTimer,
		logger:      f.logger,
		conduit:     f.conduit,
		metrics:     f.metrics.idleStateMetrics,
	}
}

func (f *headerStateFactory) CreateCollectingAvailabilityResponseState() headerSyncState {
	return &collectingHeadersAvailabilityResponsesState{
		factory:     f,
		client:      newHeaderSyncGossipClient(f.gossip, f.storage, f.logger, f.config.BlockSyncNumBlocksInBatch, f.config.NodeAddress),
		createTimer: f.createCollectTimeoutTimer,
		logger:      f.logger,
		conduit:     f.conduit,
		metrics:     f.metrics.collectingStateMetrics,
	}
}

func (f *headerStateFactory) CreateFinishedCARState(responses []*gossipmessages.HeaderAvailabilityResponseMessage) headerSyncState {
	return &finishedCHARState{
		responses: responses,
		logger:    f.logger,
		factory:   f,
		metrics:   f.metrics.finishedCollectingStateMetrics,
	}
}

func (f *headerStateFactory) CreateWaitingForChunksState(sourceNodeAddress primitives.NodeAddress) headerSyncState {
	return &waitingForHeaderChunksState{
		sourceNodeAddress: sourceNodeAddress,
		factory:           f,
		client:            newHeaderSyncGossipClient(f.gossip, f.storage, f.logger, f.config.BlockSyncNumBlocksInBatch, f.config.NodeAddress),
		createTimer:       f.createWaitForChunksTimeoutTimer,
		logger:            f.logger,
		conduit:           f.conduit,
		metrics:           f.metrics.waitingStateMetrics,
	}
}

func (f *headerStateFactory) CreateProcessingHeadersState(message *gossipmessages.HeaderSyncResponseMessage) headerSyncState {
	return &processingHeadersState{
		headers:  message,
		factory: f,
		logger:  f.logger,
		storage: f.storage,
		metrics: f.metrics.processingStateMetrics,
	}
}


func newHeadersStateMetrics(factory metric.Factory) *stateMetrics {
	return &stateMetrics{
		idleStateMetrics: idleStateMetrics{
			timeSpentInState: factory.NewLatency("HeaderSync.IdleState.Duration.Millis", 24*30*time.Hour),
			timesReset:       factory.NewGauge("HeaderSync.IdleState.ResetBackToIdle.Count"),
			timesExpired:     factory.NewGauge("HeaderSync.IdleState.StartedBlockSync.Count"),
		},
		collectingStateMetrics: collectingStateMetrics{
			timeSpentInState:                         factory.NewLatency("HeaderSync.CollectingAvailabilityResponsesState.Duration.Millis", 24*30*time.Hour),
			timesSucceededSendingAvailabilityRequest: factory.NewGauge("HeaderSync.CollectingAvailabilityResponsesState.BroadcastSendSuccess.Count"),
			timesFailedSendingAvailabilityRequest:    factory.NewGauge("HeaderSync.CollectingAvailabilityResponsesState.BroadcastSendFailure.Count"),
		},
		finishedCollectingStateMetrics: finishedCollectingStateMetrics{
			timeSpentInState:               factory.NewLatency("HeaderSync.FinishedCollectingAvailabilityResponsesState.Duration.Millis", 24*30*time.Hour),
			finishedWithNoResponsesCount:   factory.NewGauge("HeaderSync.FinishedCollectingAvailabilityResponsesState.FinishedWithNoResponses.Count"),
			finishedWithSomeResponsesCount: factory.NewGauge("HeaderSync.FinishedCollectingAvailabilityResponsesState.FinishedWithSomeResponses.Count"),
		},
		waitingStateMetrics: waitingStateMetrics{
			timeSpentInState: factory.NewLatency("HeaderSync.WaitingForBlocksState.Duration.Millis", 24*30*time.Hour),
			timesByzantine:   factory.NewGauge("HeaderSync.WaitingForBlocksState.ReceivedBlocksFromByzantineSource.Count"),
			timesSuccessful:  factory.NewGauge("HeaderSync.WaitingForBlocksState.ReceivedBlocksFromExpectedSource.Count"),
			timesTimeout:     factory.NewGauge("HeaderSync.WaitingForBlocksState.TimedOutWithoutReceivingBlocks.Count"),
		},
		processingStateMetrics: processingStateMetrics{
			timeSpentInState:       factory.NewLatency("HeaderSync.ProcessingBlocksState.Duration.Millis", 24*30*time.Hour),
			blocksRate:             factory.NewRate("HeaderSync.ProcessingBlocksState.BlocksReceived.PerSecond"),
			committedBlocks:        factory.NewGauge("HeaderSync.ProcessingBlocksState.CommittedBlocks.Count"),
			failedCommitBlocks:     factory.NewGauge("HeaderSync.ProcessingBlocksState.FailedToCommitBlocks.Count"),
			failedValidationBlocks: factory.NewGauge("HeaderSync.ProcessingBlocksState.FailedToValidateBlocks.Count"),
			lastCommittedTime:      factory.NewGauge("HeaderSync.ProcessingBlocksState.LastCommitted.TimeNano"),
		},
	}
}

// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package internodesync

import (
	"context"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/scribe/log"
	"time"
)

var HeaderLogTag = log.String("flow", "header-sync")

// this is coupled to gossip because the entire service is (block storage)
// nothing to gain right now in decoupling just the sync
type headerSyncState interface {
	name() string
	String() string
	processState(ctx context.Context) headerSyncState
}

type blockSyncConfig interface {
	NodeAddress() primitives.NodeAddress
	BlockSyncNumBlocksInBatch() uint32
	BlockSyncNoCommitInterval() time.Duration
	BlockSyncCollectResponseTimeout() time.Duration
	BlockSyncCollectChunksTimeout() time.Duration
}


// state machine passes outside events into this channel type for consumption by the currently active state instance.
// within processState.processState() all states must read from the channel eagerly!
// keeping the channel clear for new incoming events and tossing out irrelevant messages.
type headerSyncConduit chan interface{}

func (c headerSyncConduit) drainAndCheckForShutdown(ctx context.Context) bool {
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

type HeaderSync struct {
	govnr.TreeSupervisor
	logger  log.Logger
	factory *headerStateFactory
	gossip  gossiptopics.HeaderSync
	storage BlockSyncStorage

	conduit headerSyncConduit

	metrics *stateMachineMetrics

}


func newHeaderStateMachineMetrics(factory metric.Factory) *stateMachineMetrics {
	return &stateMachineMetrics{
		statesTransitioned: factory.NewGauge("HeaderSync.StateTransitions.Count"),
	}
}

func newHeaderSyncWithFactory(ctx context.Context, factory *headerStateFactory, gossip gossiptopics.HeaderSync, storage BlockSyncStorage, logger log.Logger, metricFactory metric.Factory) *HeaderSync {
	metrics := newHeaderStateMachineMetrics(metricFactory)

	hs := &HeaderSync{
		logger:  logger,
		factory: factory,
		gossip:  gossip,
		storage: storage,
		conduit: factory.conduit,
		metrics: metrics,
	}

	logger.Info("header sync init",
		log.Stringable("no-commit-timeout", hs.factory.config.BlockSyncNoCommitInterval()),
		log.Stringable("collect-responses-timeout", hs.factory.config.BlockSyncCollectResponseTimeout()),
		log.Stringable("collect-chunks-timeout", hs.factory.config.BlockSyncCollectChunksTimeout()),
		log.Uint32("batch-size", hs.factory.config.BlockSyncNumBlocksInBatch()))

	hs.Supervise(govnr.Forever(ctx, "Node sync state machine", logfields.GovnrErrorer(logger), func() {
		hs.syncLoop(ctx)
	}))

	return hs
}

func NewHeaderSync(ctx context.Context, config blockSyncConfig, gossip gossiptopics.HeaderSync, storage BlockSyncStorage, parentLogger log.Logger, metricFactory metric.Factory) *HeaderSync {
	logger := parentLogger.WithTags(HeaderLogTag)

	conduit := make(headerSyncConduit)
	return newHeaderSyncWithFactory(
		ctx,
		NewHeaderStateFactory(config, gossip, storage, conduit, logger, metricFactory),
		gossip,
		storage,
		logger,
		metricFactory,
	)
}

func (hs *HeaderSync) syncLoop(parent context.Context) {
	for currentState := hs.factory.CreateCollectingAvailabilityResponseState(); currentState != nil; {
		ctx := trace.NewContext(parent, "HeaderSync")
		hs.logger.Info("state transitioning", log.Stringable("current-state", currentState), trace.LogFieldFrom(ctx))

		currentState = currentState.processState(ctx)
		hs.metrics.statesTransitioned.Inc()
	}
}

func (hs *HeaderSync) HandleBlockCommitted(ctx context.Context) {
	logger := hs.logger.WithTags(trace.LogFieldFrom(ctx))

	select {
	case hs.conduit <- idleResetMessage{}:
	case <-ctx.Done():
		logger.Info("terminated on handle block committed", log.Error(ctx.Err()))
	}
}

func (hs *HeaderSync) HandleHeaderAvailabilityResponse(ctx context.Context, input *gossiptopics.HeaderAvailabilityResponseInput) (*gossiptopics.EmptyOutput, error) {
	logger := hs.logger.WithTags(trace.LogFieldFrom(ctx))

	select {
	case hs.conduit <- input.Message:
	case <-ctx.Done():
		logger.Info("terminated on writing new availability response",
			log.String("context-message", ctx.Err().Error()),
			log.Stringable("response-source", input.Message.Sender.SenderNodeAddress()))
	}
	return nil, nil
}

func (hs *HeaderSync) HandleBlockSyncResponse(ctx context.Context, input *gossiptopics.HeaderSyncResponseInput) (*gossiptopics.EmptyOutput, error) {
	logger := hs.logger.WithTags(trace.LogFieldFrom(ctx))

	select {
	case hs.conduit <- input.Message:
	case <-ctx.Done():
		logger.Info("terminated on writing new block chunk message",
			log.String("context-message", ctx.Err().Error()),
			log.Stringable("message-sender", input.Message.Sender.SenderNodeAddress()))
	}

	return nil, nil
}

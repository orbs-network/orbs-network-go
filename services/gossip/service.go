// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package gossip

import (
	"context"
	"fmt"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/scribe/log"
	"sync"
)

var LogTag = log.Service("gossip")

type Config interface {
	NodeAddress() primitives.NodeAddress
	VirtualChainId() primitives.VirtualChainId
}

type gossipListeners struct {
	sync.RWMutex
	transactionHandlers        []gossiptopics.TransactionRelayHandler
	leanHelixHandlers          []gossiptopics.LeanHelixHandler
	benchmarkConsensusHandlers []gossiptopics.BenchmarkConsensusHandler
	blockSyncHandlers          []gossiptopics.BlockSyncHandler
	headerSyncHandlers         []gossiptopics.HeaderSyncHandler
}

type Service struct {
	govnr.TreeSupervisor

	config          Config
	logger          log.Logger
	transport       adapter.Transport
	handlers        gossipListeners
	headerValidator *headerValidator

	messageDispatcher             *gossipMessageDispatcher
	forwarededTransactionFailures *metric.Gauge
}

func NewGossip(ctx context.Context, transport adapter.Transport, config Config, parent log.Logger, metricRegistry metric.Registry) *Service {
	logger := parent.WithTags(LogTag)
	dispatcher := newMessageDispatcher(metricRegistry, logger)
	s := &Service{
		transport:       transport,
		config:          config,
		logger:          logger,
		handlers:        gossipListeners{},
		headerValidator: newHeaderValidator(config, parent),

		messageDispatcher:             dispatcher,
		forwarededTransactionFailures: metricRegistry.NewGauge("Gossip.Topic.TransactionRelay.Errors.Count"),
	}
	transport.RegisterListener(s, s.config.NodeAddress())
	s.Supervise(dispatcher.runHandler(ctx, logger, gossipmessages.HEADER_TOPIC_TRANSACTION_RELAY, s.receivedTransactionRelayMessage))
	s.Supervise(dispatcher.runHandler(ctx, logger, gossipmessages.HEADER_TOPIC_BLOCK_SYNC, s.receivedBlockSyncMessage))
	s.Supervise(dispatcher.runHandler(ctx, logger, gossipmessages.HEADER_TOPIC_HEADER_SYNC, s.receivedHeaderSyncMessage))
	s.Supervise(dispatcher.runHandler(ctx, logger, gossipmessages.HEADER_TOPIC_LEAN_HELIX, s.receivedLeanHelixMessage))
	s.Supervise(dispatcher.runHandler(ctx, logger, gossipmessages.HEADER_TOPIC_BENCHMARK_CONSENSUS, s.receivedBenchmarkConsensusMessage))

	return s
}

func (s *Service) UpdateTopology(bgCtx context.Context, newPeers adapter.GossipPeers) {
	s.transport.UpdateTopology(bgCtx, newPeers)
}

func (s *Service) OnTransportMessageReceived(ctx context.Context, payloads [][]byte) {
	if ctx.Err() != nil {
		return
	}

	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	if len(payloads) == 0 {
		logger.Error("transport did not receive any payloads, header missing")
		return
	}
	header := gossipmessages.HeaderReader(payloads[0])
	if !header.IsValid() {
		logger.Error("transport header is corrupt", log.Bytes("header", payloads[0]))
		return
	}

	if err := s.headerValidator.validateMessageHeader(header); err != nil {
		logger.Error("dropping a received message that isn't valid", log.Error(err), log.Stringable("message-header", header))
		return
	}

	s.messageDispatcher.dispatch(ctx, logger, header, payloads[1:])
}

func (s *Service) String() string {
	return fmt.Sprintf("Gossip service for node %s: %p", s.config.NodeAddress(), s)
}

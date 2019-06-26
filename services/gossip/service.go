// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package gossip

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
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
}

type service struct {
	config          Config
	logger          log.Logger
	transport       adapter.Transport
	handlers        gossipListeners
	headerValidator *headerValidator

	transactionRelayChannel   chan gossipMessage
	blockSyncChannel          chan gossipMessage
	leanHelixChannel          chan gossipMessage
	benchmarkConsensusChannel chan gossipMessage
}

type gossipMessage struct {
	header   *gossipmessages.Header
	payloads [][]byte
}

func NewGossip(transport adapter.Transport, config Config, parent log.Logger) services.Gossip {
	logger := parent.WithTags(LogTag)
	s := &service{
		transport:       transport,
		config:          config,
		logger:          logger,
		handlers:        gossipListeners{},
		headerValidator: newHeaderValidator(config, parent),

		transactionRelayChannel:   make(chan gossipMessage),
		blockSyncChannel:          make(chan gossipMessage),
		leanHelixChannel:          make(chan gossipMessage),
		benchmarkConsensusChannel: make(chan gossipMessage),
	}
	transport.RegisterListener(s, s.config.NodeAddress())

	ctx := context.TODO()
	runGossipTopicHandler(ctx, logger, s.transactionRelayChannel, s.receivedTransactionRelayMessage)
	runGossipTopicHandler(ctx, logger, s.blockSyncChannel, s.receivedBlockSyncMessage)
	runGossipTopicHandler(ctx, logger, s.leanHelixChannel, s.receivedLeanHelixMessage)
	runGossipTopicHandler(ctx, logger, s.benchmarkConsensusChannel, s.receivedBenchmarkConsensusMessage)

	return s
}

func runGossipTopicHandler(ctx context.Context, logger log.Logger, ch chan gossipMessage, handler func(ctx context.Context, header *gossipmessages.Header, payloads [][]byte)) supervised.ContextEndedChan {
	return supervised.GoForever(ctx, logger, func() {
		for {
			select {
			case <-ctx.Done():
				return
			case message := <-ch:
				handler(ctx, message.header, message.payloads)
			}
		}
	})
}

func (s *service) OnTransportMessageReceived(ctx context.Context, payloads [][]byte) {
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

	logger.Info("transport message received", log.Stringable("header", header), log.String("gossip-topic", header.StringTopic()))
	switch header.Topic() {
	case gossipmessages.HEADER_TOPIC_TRANSACTION_RELAY:
		s.transactionRelayChannel <- gossipMessage{header: header, payloads: payloads[1:]} //TODO should the channel have *gossipMessage as type
	case gossipmessages.HEADER_TOPIC_LEAN_HELIX:
		s.leanHelixChannel <- gossipMessage{header: header, payloads: payloads[1:]} //TODO should the channel have *gossipMessage as type
	case gossipmessages.HEADER_TOPIC_BENCHMARK_CONSENSUS:
		s.benchmarkConsensusChannel <- gossipMessage{header: header, payloads: payloads[1:]} //TODO should the channel have *gossipMessage as type
	case gossipmessages.HEADER_TOPIC_BLOCK_SYNC:
		s.blockSyncChannel <- gossipMessage{header: header, payloads: payloads[1:]} //TODO should the channel have *gossipMessage as type
	}
}

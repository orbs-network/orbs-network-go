// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package gossip

import (
	"context"
	"fmt"
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

	messageDispatcher gossipMessageDispatcher
}

type gossipMessage struct {
	header   *gossipmessages.Header
	payloads [][]byte
}

func NewGossip(ctx context.Context, transport adapter.Transport, config Config, parent log.Logger) services.Gossip {
	logger := parent.WithTags(LogTag)
	s := &service{
		transport:       transport,
		config:          config,
		logger:          logger,
		handlers:        gossipListeners{},
		headerValidator: newHeaderValidator(config, parent),

		messageDispatcher: makeMessageDispatcher(),
	}
	transport.RegisterListener(s, s.config.NodeAddress())

	s.messageDispatcher.runHandler(ctx, logger, gossipmessages.HEADER_TOPIC_TRANSACTION_RELAY, s.receivedTransactionRelayMessage)
	s.messageDispatcher.runHandler(ctx, logger, gossipmessages.HEADER_TOPIC_BLOCK_SYNC, s.receivedBlockSyncMessage)
	s.messageDispatcher.runHandler(ctx, logger, gossipmessages.HEADER_TOPIC_LEAN_HELIX, s.receivedLeanHelixMessage)
	s.messageDispatcher.runHandler(ctx, logger, gossipmessages.HEADER_TOPIC_BENCHMARK_CONSENSUS, s.receivedBenchmarkConsensusMessage)

	return s
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
	s.messageDispatcher.dispatch(logger, header, payloads[1:])
}

type gossipMessageDispatcher map[gossipmessages.HeaderTopic]chan gossipMessage

func makeMessageDispatcher() (d gossipMessageDispatcher) {
	d = make(gossipMessageDispatcher)
	d[gossipmessages.HEADER_TOPIC_TRANSACTION_RELAY] = make(chan gossipMessage)
	d[gossipmessages.HEADER_TOPIC_BLOCK_SYNC] = make(chan gossipMessage)
	d[gossipmessages.HEADER_TOPIC_LEAN_HELIX] = make(chan gossipMessage)
	d[gossipmessages.HEADER_TOPIC_BENCHMARK_CONSENSUS] = make(chan gossipMessage)
	return
}

func (d gossipMessageDispatcher) dispatch(logger log.Logger, header *gossipmessages.Header, payloads [][]byte) {
	ch := d[header.Topic()]
	if ch == nil {
		logger.Error("no message channel for topic", log.Int("topic", int(header.Topic())))
		return
	}

	ch <- gossipMessage{header: header, payloads: payloads} //TODO should the channel have *gossipMessage as type?
}

func (d gossipMessageDispatcher) runHandler(ctx context.Context, logger log.Logger, topic gossipmessages.HeaderTopic, handler func(ctx context.Context, header *gossipmessages.Header, payloads [][]byte)) {
	ch := d[topic]
	if ch == nil {
		panic(fmt.Sprintf("no message channel for topic %d", topic))
	} else {
		supervised.GoForever(ctx, logger, func() {
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

}

package gossip

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"sync"
)

var LogTag = log.Service("gossip")

type Config interface {
	NodeAddress() primitives.NodeAddress
}

type gossipListeners struct {
	sync.RWMutex
	transactionHandlers        []gossiptopics.TransactionRelayHandler
	leanHelixHandlers          []gossiptopics.LeanHelixHandler
	benchmarkConsensusHandlers []gossiptopics.BenchmarkConsensusHandler
	blockSyncHandlers          []gossiptopics.BlockSyncHandler
}

type service struct {
	config    Config
	logger    log.BasicLogger
	transport adapter.Transport
	handlers  gossipListeners
}

func NewGossip(transport adapter.Transport, config Config, logger log.BasicLogger) services.Gossip {
	s := &service{
		transport: transport,
		config:    config,
		logger:    logger.WithTags(LogTag),
		handlers:  gossipListeners{},
	}
	transport.RegisterListener(s, s.config.NodeAddress())
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
	logger.Info("transport message received", log.Stringable("header", header), log.String("gossip-topic", header.StringTopic()))
	switch header.Topic() {
	case gossipmessages.HEADER_TOPIC_TRANSACTION_RELAY:
		s.receivedTransactionRelayMessage(ctx, header, payloads[1:])
	case gossipmessages.HEADER_TOPIC_LEAN_HELIX:
		s.receivedLeanHelixMessage(ctx, header, payloads[1:])
	case gossipmessages.HEADER_TOPIC_BENCHMARK_CONSENSUS:
		s.receivedBenchmarkConsensusMessage(ctx, header, payloads[1:])
	case gossipmessages.HEADER_TOPIC_BLOCK_SYNC:
		s.receivedBlockSyncMessage(ctx, header, payloads[1:])
	}
}

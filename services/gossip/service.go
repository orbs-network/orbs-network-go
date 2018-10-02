package gossip

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

var LogTag = log.Service("gossip")

type Config interface {
	NodePublicKey() primitives.Ed25519PublicKey
}

type service struct {
	config                     Config
	logger                     log.BasicLogger
	transport                  adapter.Transport
	transactionHandlers        []gossiptopics.TransactionRelayHandler
	leanHelixHandlers          []gossiptopics.LeanHelixHandler
	benchmarkConsensusHandlers []gossiptopics.BenchmarkConsensusHandler
	blockSyncHandlers          []gossiptopics.BlockSyncHandler
}

func NewGossip(transport adapter.Transport, config Config, logger log.BasicLogger) services.Gossip {
	s := &service{
		transport: transport,
		config:    config,
		logger:    logger.WithTags(LogTag),
	}
	transport.RegisterListener(s, s.config.NodePublicKey())
	return s
}

func (s *service) OnTransportMessageReceived(payloads [][]byte) {
	if len(payloads) == 0 {
		s.logger.Error("transport did not receive any payloads, header missing")
		return
	}
	header := gossipmessages.HeaderReader(payloads[0])
	if !header.IsValid() {
		s.logger.Error("transport header is corrupt", log.Bytes("header", payloads[0]))
		return
	}
	s.logger.Info("transport message received", log.Stringable("header", header))
	switch header.Topic() {
	case gossipmessages.HEADER_TOPIC_TRANSACTION_RELAY:
		s.receivedTransactionRelayMessage(header, payloads[1:])
	case gossipmessages.HEADER_TOPIC_LEAN_HELIX:
		s.receivedLeanHelixMessage(header, payloads[1:])
	case gossipmessages.HEADER_TOPIC_BENCHMARK_CONSENSUS:
		s.receivedBenchmarkConsensusMessage(header, payloads[1:])
	case gossipmessages.HEADER_TOPIC_BLOCK_SYNC:
		s.receivedBlockSyncMessage(header, payloads[1:])
	}
}

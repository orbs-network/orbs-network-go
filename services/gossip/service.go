package gossip

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

type Config interface {
	NodePublicKey() primitives.Ed25519Pkey
}

type service struct {
	config                     Config
	reporting                  instrumentation.Reporting
	transport                  adapter.Transport
	transactionHandlers        []gossiptopics.TransactionRelayHandler
	leanHelixHandlers          []gossiptopics.LeanHelixHandler
	benchmarkConsensusHandlers []gossiptopics.BenchmarkConsensusHandler
}

func NewGossip(transport adapter.Transport, config Config, reporting instrumentation.Reporting) services.Gossip {
	s := &service{
		transport: transport,
		config:    config,
		reporting: reporting,
	}
	transport.RegisterListener(s, s.config.NodePublicKey())
	return s
}

func (s *service) OnTransportMessageReceived(payloads [][]byte) {
	if len(payloads) == 0 {
		s.reporting.Error(&adapter.ErrCorruptData{})
		return
	}
	header := gossipmessages.HeaderReader(payloads[0])
	if !header.IsValid() {
		s.reporting.Error(&ErrCorruptHeader{payloads[0]})
		return
	}
	s.reporting.Info(fmt.Sprintf("Gossip: OnTransportMessageReceived: %s", header))
	switch header.Topic() {
	case gossipmessages.HEADER_TOPIC_TRANSACTION_RELAY:
		s.receivedTransactionRelayMessage(header, payloads[1:])
	case gossipmessages.HEADER_TOPIC_LEAN_HELIX:
		s.receivedLeanHelixMessage(header, payloads[1:])
	case gossipmessages.HEADER_TOPIC_BENCHMARK_CONSENSUS:
		s.receivedBenchmarkConsensusMessage(header, payloads[1:])
	}
}

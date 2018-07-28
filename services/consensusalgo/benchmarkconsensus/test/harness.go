package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/consensusalgo/benchmarkconsensus"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"testing"
)

type harness struct {
	gossip           *gossiptopics.MockBenchmarkConsensus
	blockStorage     *services.MockBlockStorage
	consensusContext *services.MockConsensusContext
	reporting        instrumentation.Reporting
	config           benchmarkconsensus.Config
	service          services.ConsensusAlgoBenchmark
}

func newHarness(
	isLeader bool,
) *harness {

	leaderKeyPair := keys.Ed25519KeyPairForTests(0)
	nodeKeyPair := leaderKeyPair
	if !isLeader {
		nodeKeyPair = keys.Ed25519KeyPairForTests(1)
	}

	config := config.NewHardCodedConfig(
		5,
		nodeKeyPair.PublicKey(),
		nodeKeyPair.PrivateKey(),
		leaderKeyPair.PublicKey(),
		consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS,
		5,
	)

	log := instrumentation.NewStdoutLog()

	gossip := &gossiptopics.MockBenchmarkConsensus{}
	gossip.When("RegisterBenchmarkConsensusHandler", mock.Any).Return().Times(1)

	blockStorage := &services.MockBlockStorage{}
	blockStorage.When("RegisterConsensusBlocksHandler", mock.Any).Return().Times(1)

	consensusContext := &services.MockConsensusContext{}

	return &harness{
		gossip:           gossip,
		blockStorage:     blockStorage,
		consensusContext: consensusContext,
		reporting:        log,
		config:           config,
		service:          nil,
	}
}

func (h *harness) createService(ctx context.Context) {
	h.service = benchmarkconsensus.NewBenchmarkConsensusAlgo(
		ctx,
		h.gossip,
		h.blockStorage,
		h.consensusContext,
		h.reporting,
		h.config,
	)
}

func nonLeaderKeyPair() (primitives.Ed25519PublicKey, primitives.Ed25519PrivateKey) {
	keyPair := keys.Ed25519KeyPairForTests(3)
	return keyPair.PublicKey(), keyPair.PrivateKey()
}

func (h *harness) verifyHandlerRegistrations(t *testing.T) {
	ok, err := h.gossip.Verify()
	if !ok {
		t.Fatal("Did not register with Gossip:", err)
	}
	ok, err = h.blockStorage.Verify()
	if !ok {
		t.Fatal("Did not register with BlockStorage:", err)
	}
}

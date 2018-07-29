package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/consensusalgo/benchmarkconsensus"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"testing"
)

const networkSize = 5

type harness struct {
	gossip           *gossiptopics.MockBenchmarkConsensus
	blockStorage     *services.MockBlockStorage
	consensusContext *services.MockConsensusContext
	reporting        instrumentation.Reporting
	config           benchmarkconsensus.Config
	service          services.ConsensusAlgoBenchmark
}

func leaderKeyPair() (primitives.Ed25519PublicKey, primitives.Ed25519PrivateKey) {
	keyPair := keys.Ed25519KeyPairForTests(0)
	return keyPair.PublicKey(), keyPair.PrivateKey()
}

func nonLeaderKeyPair() (primitives.Ed25519PublicKey, primitives.Ed25519PrivateKey) {
	keyPair := keys.Ed25519KeyPairForTests(1)
	return keyPair.PublicKey(), keyPair.PrivateKey()
}

func otherNonLeaderKeyPair() (primitives.Ed25519PublicKey, primitives.Ed25519PrivateKey) {
	keyPair := keys.Ed25519KeyPairForTests(2)
	return keyPair.PublicKey(), keyPair.PrivateKey()
}

func newHarness(
	isLeader bool,
) *harness {

	federationNodes := make(map[string]config.FederationNode)
	for i := 0; i < networkSize; i++ {
		publicKey := keys.Ed25519KeyPairForTests(i).PublicKey()
		federationNodes[publicKey.KeyForMap()] = config.NewHardCodedFederationNode(publicKey)
	}

	leaderPublicKey, leaderPrivateKey := leaderKeyPair()
	nodePublicKey, nodePrivateKey := leaderPublicKey, leaderPrivateKey
	if !isLeader {
		nodePublicKey, nodePrivateKey = nonLeaderKeyPair()
	}

	config := config.NewHardCodedConfig(
		federationNodes,
		nodePublicKey,
		nodePrivateKey,
		leaderPublicKey,
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

func (h *harness) handleBlockConsensus(blockPair *protocol.BlockPairContainer, prevCommitted *protocol.BlockPairContainer) error {
	_, err := h.service.HandleBlockConsensus(&handlers.HandleBlockConsensusInput{
		BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
		BlockPair:              blockPair,
		PrevCommittedBlockPair: prevCommitted,
	})
	return err
}

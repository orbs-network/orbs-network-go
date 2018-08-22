package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/consensusalgo/benchmarkconsensus"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"os"
	"testing"
	"time"
)

const networkSize = 5

type harness struct {
	gossip           *gossiptopics.MockBenchmarkConsensus
	blockStorage     *services.MockBlockStorage
	consensusContext *services.MockConsensusContext
	reporting        log.BasicLogger
	config           benchmarkconsensus.Config
	service          services.ConsensusAlgoBenchmark
}

func leaderKeyPair() *keys.Ed25519KeyPair {
	return keys.Ed25519KeyPairForTests(0)
}

func nonLeaderKeyPair() *keys.Ed25519KeyPair {
	return keys.Ed25519KeyPairForTests(1)
}

func otherNonLeaderKeyPair() *keys.Ed25519KeyPair {
	return keys.Ed25519KeyPairForTests(2)
}

func newHarness(
	isLeader bool,
) *harness {

	federationNodes := make(map[string]config.FederationNode)
	for i := 0; i < networkSize; i++ {
		publicKey := keys.Ed25519KeyPairForTests(i).PublicKey()
		federationNodes[publicKey.KeyForMap()] = config.NewHardCodedFederationNode(publicKey)
	}

	nodeKeyPair := leaderKeyPair()
	if !isLeader {
		nodeKeyPair = nonLeaderKeyPair()
	}

	cfg := config.ForAcceptanceTests(
		federationNodes,
		nodeKeyPair.PublicKey(),
		nodeKeyPair.PrivateKey(),
		leaderKeyPair().PublicKey(),
		consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS,
	)

	cfg.SetDuration(config.BENCHMARK_CONSENSUS_RETRY_INTERVAL_MILLIS, 5*time.Millisecond)

	log := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))

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
		config:           cfg,
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

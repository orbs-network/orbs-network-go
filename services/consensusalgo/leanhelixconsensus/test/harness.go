package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/consensusalgo/leanhelixconsensus"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"testing"
)

const NETWORK_SIZE = 5

type harness struct {
	gossip           *gossiptopics.MockLeanHelix
	blockStorage     *services.MockBlockStorage
	consensusContext *services.MockConsensusContext
	reporting        log.BasicLogger
	config           leanhelixconsensus.Config
	service          services.ConsensusAlgoLeanHelix
	registry         metric.Registry
}

func leaderKeyPair() *keys.Ed25519KeyPair {
	return keys.Ed25519KeyPairForTests(0)
}

func nonLeaderKeyPair() *keys.Ed25519KeyPair {
	return keys.Ed25519KeyPairForTests(1)
}

// TODO Uncomment when used
//func newHarness(
//	isLeader bool,
//) *harness {
//
//	federationNodes := make(map[string]config.FederationNode)
//	for i := 0; i < NETWORK_SIZE; i++ {
//		publicKey := keys.Ed25519KeyPairForTests(i).PublicKey()
//		federationNodes[publicKey.KeyForMap()] = config.NewHardCodedFederationNode(publicKey)
//	}
//
//	nodeKeyPair := leaderKeyPair()
//	if !isLeader {
//		nodeKeyPair = nonLeaderKeyPair()
//	}
//
//	cfg := config.ForAcceptanceTests(
//		federationNodes,
//		make(map[string]config.GossipPeer),
//		nodeKeyPair.PublicKey(),
//		nodeKeyPair.PrivateKey(),
//		leaderKeyPair().PublicKey(),
//		consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX,
//		1,
//	)
//
//	cfg.SetDuration(config.LEAN_HELIX_CONSENSUS_RETRY_INTERVAL, 5*time.Millisecond)
//
//	log := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
//
//	gossip := &gossiptopics.MockLeanHelix{}
//	gossip.When("RegisterBenchmarkConsensusHandler", mock.Any).Return().Times(1)
//
//	blockStorage := &services.MockBlockStorage{}
//	blockStorage.When("RegisterConsensusBlocksHandler", mock.Any).Return().Times(1)
//
//	consensusContext := &services.MockConsensusContext{}
//
//	return &harness{
//		gossip:           gossip,
//		blockStorage:     blockStorage,
//		consensusContext: consensusContext,
//		reporting:        log,
//		config:           cfg,
//		service:          nil,
//		registry:         metric.NewRegistry(),
//	}
//}

func (h *harness) createService(ctx context.Context) {
	h.service = leanhelixconsensus.NewLeanHelixConsensusAlgo(
		ctx,
		h.gossip,
		h.blockStorage,
		h.consensusContext,
		h.reporting,
		h.config,
		h.registry,
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

func (h *harness) handleBlockConsensus(ctx context.Context, mode handlers.HandleBlockConsensusMode, blockPair *protocol.BlockPairContainer, prevCommitted *protocol.BlockPairContainer) error {
	_, err := h.service.HandleBlockConsensus(ctx, &handlers.HandleBlockConsensusInput{
		Mode:                   mode,
		BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
		BlockPair:              blockPair,
		PrevCommittedBlockPair: prevCommitted,
	})
	return err
}

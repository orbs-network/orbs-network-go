package test

import (
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/consensusalgo/benchmarkconsensus"
	testInstrumentation "github.com/orbs-network/orbs-network-go/test/harness/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

type context struct {
	gossip           *gossiptopics.MockBenchmarkConsensus
	blockStorage     *services.MockBlockStorage
	consensusContext *services.MockConsensusContext
	reporting        instrumentation.Reporting
	config           benchmarkconsensus.Config
	service          services.ConsensusAlgoBenchmark
}

func newContext(
	isLeader bool,
) *context {

	leaderPublicKey := []byte{0x01}
	nodePublicKey := leaderPublicKey
	if !isLeader {
		nodePublicKey = []byte{0x02}
	}

	config := config.NewHardCodedConfig(
		5,
		nodePublicKey,
		leaderPublicKey,
		consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS,
	)

	log := testInstrumentation.NewBufferedLog("BenchmarkConsensus")

	gossip := &gossiptopics.MockBenchmarkConsensus{}
	gossip.When("RegisterBenchmarkConsensusHandler", mock.Any).Return().Times(1)

	blockStorage := &services.MockBlockStorage{}
	blockStorage.When("RegisterConsensusBlocksHandler", mock.Any).Return().Times(1)

	consensusContext := &services.MockConsensusContext{}

	return &context{
		gossip:           gossip,
		blockStorage:     blockStorage,
		consensusContext: consensusContext,
		reporting:        log,
		config:           config,
		service:          nil,
	}
}

func (c *context) createService() {
	c.service = benchmarkconsensus.NewBenchmarkConsensusAlgo(
		c.gossip,
		c.blockStorage,
		c.consensusContext,
		c.reporting,
		c.config,
	)
}

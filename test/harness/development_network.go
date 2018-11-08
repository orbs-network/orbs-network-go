package harness

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	gossipAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
)

func NewDevelopmentNetwork(logger log.BasicLogger) *inProcessNetwork {
	numNodes := 2
	consensusAlgo := consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS
	logger.Info("creating development network")
	description := fmt.Sprintf("network with %d nodes running %s", numNodes, consensusAlgo)

	leaderKeyPair := keys.Ed25519KeyPairForTests(0)

	federationNodes := make(map[string]config.FederationNode)
	gossipPeers := make(map[string]config.GossipPeer)
	for i := 0; i < int(numNodes); i++ {
		publicKey := keys.Ed25519KeyPairForTests(i).PublicKey()
		federationNodes[publicKey.KeyForMap()] = config.NewHardCodedFederationNode(publicKey)
		gossipPeers[publicKey.KeyForMap()] = config.NewHardCodedGossipPeer(0, "")
	}

	sharedTamperingTransport := gossipAdapter.NewTamperingTransport(logger, federationNodes)

	nodes := make([]*networkNode, numNodes)
	for i := range nodes {
		node := &networkNode{}
		node.index = i
		nodeKeyPair := keys.Ed25519KeyPairForTests(i)
		node.name = fmt.Sprintf("%s", nodeKeyPair.PublicKey()[:3])

		node.config = config.ForGamma(
			federationNodes,
			gossipPeers,
			nodeKeyPair.PublicKey(),
			nodeKeyPair.PrivateKey(),
			leaderKeyPair.PublicKey(),
			consensusAlgo,
		)

		node.statePersistence = stateStorageAdapter.NewTamperingStatePersistence()
		node.blockPersistence = blockStorageAdapter.NewInMemoryBlockPersistence()
		node.nativeCompiler = nativeProcessorAdapter.NewNativeCompiler(node.config, logger)

		node.metricRegistry = metric.NewRegistry()

		nodes[i] = node
	}

	return &inProcessNetwork{
		nodes:           nodes,
		gossipTransport: sharedTamperingTransport,
		description:     description,
		testLogger:      logger,
	}

	// must call network.Start(ctx) to actually start the nodes in the network
}

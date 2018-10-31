package harness

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	gossipAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/processor/native/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
)

func NewAcceptanceTestNetwork(numNodes uint32, testLogger log.BasicLogger, consensusAlgo consensus.ConsensusAlgoType, maxTxPerBlock uint32) *inProcessNetwork {


	testLogger.Info("===========================================================================")
	testLogger.Info("creating acceptance test network", log.String("consensus", consensusAlgo.String()), log.Uint32("num-nodes", numNodes))
	description := fmt.Sprintf("network with %d nodes running %s", numNodes, consensusAlgo)

	sharedTamperingTransport := gossipAdapter.NewTamperingTransport(testLogger)
	leaderKeyPair := keys.Ed25519KeyPairForTests(0)

	federationNodes := make(map[string]config.FederationNode)
	gossipPeers := make(map[string]config.GossipPeer)
	for i := 0; i < int(numNodes); i++ {
		publicKey := keys.Ed25519KeyPairForTests(i).PublicKey()
		federationNodes[publicKey.KeyForMap()] = config.NewHardCodedFederationNode(publicKey)
		gossipPeers[publicKey.KeyForMap()] = config.NewHardCodedGossipPeer(0, "")
	}

	nodes := make([]*networkNode, numNodes)
	for i := range nodes {
		node := &networkNode{}
		node.index = i
		nodeKeyPair := keys.Ed25519KeyPairForTests(i)
		node.name = fmt.Sprintf("%s", nodeKeyPair.PublicKey()[:3])

		node.config = config.ForAcceptanceTests(
			federationNodes,
			gossipPeers,
			nodeKeyPair.PublicKey(),
			nodeKeyPair.PrivateKey(),
			leaderKeyPair.PublicKey(),
			consensusAlgo,
			maxTxPerBlock,
		)

		node.statePersistence = stateStorageAdapter.NewTamperingStatePersistence()
		node.blockPersistence = blockStorageAdapter.NewInMemoryBlockPersistence()
		node.nativeCompiler = nativeProcessorAdapter.NewFakeCompiler()

		node.metricRegistry = metric.NewRegistry()

		nodes[i] = node
	}

	return &inProcessNetwork{
		nodes:           nodes,
		gossipTransport: sharedTamperingTransport,
		description:     description,
		testLogger:      testLogger,
	}

	// must call network.StartNodes(ctx) to actually start the nodes in the network
}

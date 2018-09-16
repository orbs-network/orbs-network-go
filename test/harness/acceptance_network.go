package harness

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	gossipAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/processor/native/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"os"
)

func NewAcceptanceTestNetwork(numNodes uint32, consensusAlgo consensus.ConsensusAlgoType, testId string) *inProcessNetwork {
	testLogger := log.GetLogger(log.String("_test-id", testId)).WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	testLogger.Info("===========================================================================")
	testLogger.Info("creating acceptance test network", log.String("consensus", consensusAlgo.String()), log.Uint32("num-nodes", numNodes))
	description := fmt.Sprintf("network with %d nodes running %s", numNodes, consensusAlgo)

	sharedTamperingTransport := gossipAdapter.NewTamperingTransport()
	leaderKeyPair := keys.Ed25519KeyPairForTests(0)

	federationNodes := make(map[string]config.FederationNode)
	for i := 0; i < int(numNodes); i++ {
		publicKey := keys.Ed25519KeyPairForTests(i).PublicKey()
		federationNodes[publicKey.KeyForMap()] = config.NewHardCodedFederationNode(publicKey)
	}

	nodes := make([]*networkNode, numNodes)
	for i := range nodes {
		node := &networkNode{}
		node.index = i
		nodeKeyPair := keys.Ed25519KeyPairForTests(i)
		node.name = fmt.Sprintf("%s", nodeKeyPair.PublicKey()[:3])

		node.config = config.ForAcceptanceTests(
			federationNodes,
			nodeKeyPair.PublicKey(),
			nodeKeyPair.PrivateKey(),
			leaderKeyPair.PublicKey(),
			consensusAlgo,
		)

		node.statePersistence = stateStorageAdapter.NewTamperingStatePersistence()
		node.blockPersistence = blockStorageAdapter.NewInMemoryBlockPersistence()
		node.nativeCompiler = nativeProcessorAdapter.NewFakeCompiler()

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

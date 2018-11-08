package harness

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/harness/contracts"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	gossipAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/processor/native/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
)

type InProcessTestNetwork interface {
	InProcessNetwork
	GetBenchmarkTokenContract() contracts.BenchmarkTokenClient
	TransportTamperer() gossipAdapter.Tamperer
	Description() string
	BlockPersistence(nodeIndex int) blockStorageAdapter.InMemoryBlockPersistence
	DumpState()
	WaitForTransactionInState(ctx context.Context, nodeIndex int, txhash primitives.Sha256)
	Size() int
	MetricsString(nodeIndex int) string
}

func NewAcceptanceTestNetwork(numNodes uint32, testLogger log.BasicLogger, consensusAlgo consensus.ConsensusAlgoType, maxTxPerBlock uint32) *acceptanceNetwork {

	testLogger.Info("===========================================================================")
	testLogger.Info("creating acceptance test network", log.String("consensus", consensusAlgo.String()), log.Uint32("num-nodes", numNodes))
	description := fmt.Sprintf("network with %d nodes running %s", numNodes, consensusAlgo)

	leaderKeyPair := keys.Ed25519KeyPairForTests(0)

	federationNodes := make(map[string]config.FederationNode)
	gossipPeers := make(map[string]config.GossipPeer)
	for i := 0; i < int(numNodes); i++ {
		publicKey := keys.Ed25519KeyPairForTests(i).PublicKey()
		federationNodes[publicKey.KeyForMap()] = config.NewHardCodedFederationNode(publicKey)
		gossipPeers[publicKey.KeyForMap()] = config.NewHardCodedGossipPeer(0, "")
	}

	sharedTamperingTransport := gossipAdapter.NewTamperingTransport(testLogger, federationNodes)

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

	return &acceptanceNetwork{
		inProcessNetwork:   inProcessNetwork{nodes: nodes, logger: testLogger, transport: sharedTamperingTransport},
		tamperingTransport: sharedTamperingTransport,
		description:        description,
	}

	// must call network.Start(ctx) to actually start the nodes in the network
}

type acceptanceNetwork struct {
	inProcessNetwork

	tamperingTransport *gossipAdapter.TamperingTransport
	description        string
}

func (n *acceptanceNetwork) Start(ctx context.Context) InProcessNetwork {
	n.tamperingTransport.Start(ctx)
	n.createAndStartNodes(ctx) // needs to start first so that nodes can register their listeners to it
	return n
}

func (n *acceptanceNetwork) WaitForTransactionInState(ctx context.Context, nodeIndex int, txhash primitives.Sha256) {
	n.nodes[nodeIndex].WaitForTransactionInState(ctx, txhash)
}

func (n *acceptanceNetwork) MetricsString(i int) string {
	return n.nodes[i].metricRegistry.String()
}

func (n *acceptanceNetwork) Description() string {
	return n.description
}

func (n *acceptanceNetwork) TransportTamperer() gossipAdapter.Tamperer {
	return n.tamperingTransport
}

func (n *acceptanceNetwork) BlockPersistence(nodeIndex int) blockStorageAdapter.InMemoryBlockPersistence {
	return n.nodes[nodeIndex].blockPersistence
}

func (n *acceptanceNetwork) GetBenchmarkTokenContract() contracts.BenchmarkTokenClient {
	return contracts.NewContractClient(n.nodesAsContractAPIProviders(), n.logger)
}

func (n *acceptanceNetwork) DumpState() {
	for i := range n.nodes {
		n.logger.Info("state dump", log.Int("node", i), log.String("data", n.nodes[i].statePersistence.Dump()))
	}
}

func (n *acceptanceNetwork) Size() int {
	return len(n.nodes)
}

package harness

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/inprocess"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/harness/contracts"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	gossipAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/processor/native/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
)

type TestNetworkDriver interface {
	inprocess.NetworkDriver
	GetBenchmarkTokenContract() contracts.BenchmarkTokenClient
	TransportTamperer() gossipAdapter.Tamperer
	Description() string
	BlockPersistence(nodeIndex int) blockStorageAdapter.InMemoryBlockPersistence
	DumpState()
	WaitForTransactionInState(ctx context.Context, nodeIndex int, txhash primitives.Sha256)
	Size() int
}

func NewAcceptanceTestNetwork(numNodes uint32, testLogger log.BasicLogger, consensusAlgo consensus.ConsensusAlgoType, maxTxPerBlock uint32) *acceptanceNetwork {

	testLogger.Info("===========================================================================")
	testLogger.Info("creating acceptance test network", log.String("consensus", consensusAlgo.String()), log.Uint32("num-nodes", numNodes))
	description := fmt.Sprintf("network with %d nodes running %s", numNodes, consensusAlgo)

	leaderKeyPair := testKeys.Ed25519KeyPairForTests(0)

	federationNodes := make(map[string]config.FederationNode)
	gossipPeers := make(map[string]config.GossipPeer)
	for i := 0; i < int(numNodes); i++ {
		publicKey := testKeys.Ed25519KeyPairForTests(i).PublicKey()
		federationNodes[publicKey.KeyForMap()] = config.NewHardCodedFederationNode(publicKey)
		gossipPeers[publicKey.KeyForMap()] = config.NewHardCodedGossipPeer(0, "")
	}

	sharedTamperingTransport := gossipAdapter.NewTamperingTransport(testLogger, federationNodes)

	nodes := make([]*inprocess.Node, numNodes)
	for i := range nodes {
		keyPair := testKeys.Ed25519KeyPairForTests(i)
		cfg := config.ForAcceptanceTests(
			federationNodes,
			gossipPeers,
			keyPair.PublicKey(),
			keyPair.PrivateKey(),
			leaderKeyPair.PublicKey(),
			consensusAlgo,
			maxTxPerBlock,
		)
		nodes[i] = inprocess.NewNode(i, keyPair, cfg, nativeProcessorAdapter.NewFakeCompiler())
	}

	return &acceptanceNetwork{
		Network:            inprocess.Network{Nodes: nodes, Logger: testLogger, Transport: sharedTamperingTransport},
		tamperingTransport: sharedTamperingTransport,
		description:        description,
	}

	// must call network.Start(ctx) to actually start the nodes in the network
}

type acceptanceNetwork struct {
	inprocess.Network

	tamperingTransport *gossipAdapter.TamperingTransport
	description        string
}

func (n *acceptanceNetwork) Start(ctx context.Context) inprocess.NetworkDriver {
	n.tamperingTransport.Start(ctx)
	n.CreateAndStartNodes(ctx) // needs to start first so that nodes can register their listeners to it
	return n
}

func (n *acceptanceNetwork) WaitForTransactionInState(ctx context.Context, nodeIndex int, txhash primitives.Sha256) {
	n.Nodes[nodeIndex].WaitForTransactionInState(ctx, txhash)
}

func (n *acceptanceNetwork) Description() string {
	return n.description
}

func (n *acceptanceNetwork) TransportTamperer() gossipAdapter.Tamperer {
	return n.tamperingTransport
}

func (n *acceptanceNetwork) BlockPersistence(nodeIndex int) blockStorageAdapter.InMemoryBlockPersistence {
	return n.GetBlockPersistence(nodeIndex)
}

func (n *acceptanceNetwork) GetBenchmarkTokenContract() contracts.BenchmarkTokenClient {
	return contracts.NewContractClient(n.GetAPIProviders(), n.Logger)
}

func (n *acceptanceNetwork) DumpState() {
	for i := range n.Nodes {
		n.Logger.Info("state dump", log.Int("node", i), log.String("data", n.GetStatePersistence(i).Dump()))
	}
}

func (n *acceptanceNetwork) Size() int {
	return len(n.Nodes)
}


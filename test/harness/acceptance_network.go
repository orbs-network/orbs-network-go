package harness

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-network-go/bootstrap/inmemory"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	gossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/harness/contracts"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	testGossipAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/processor/native/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
)

type TestNetworkDriver interface {
	inmemory.NetworkDriver
	GetBenchmarkTokenContract() contracts.BenchmarkTokenClient
	TransportTamperer() testGossipAdapter.Tamperer
	Description() string
	BlockPersistence(nodeIndex int) blockStorageAdapter.InMemoryBlockPersistence
	StatePersistence(nodeIndex int) stateStorageAdapter.TamperingStatePersistence
	DumpState()
	WaitForTransactionInNodeState(ctx context.Context, txhash primitives.Sha256, nodeIndex int,)
	MockContract(fakeContractInfo *sdk.ContractInfo, code string)
}

func NewAcceptanceTestNetwork(ctx context.Context, numNodes int, testLogger log.BasicLogger, consensusAlgo consensus.ConsensusAlgoType, maxTxPerBlock uint32) *acceptanceNetwork {

	testLogger.Info("===========================================================================")
	testLogger.Info("creating acceptance test network", log.String("consensus", consensusAlgo.String()), log.Int("num-nodes", numNodes))
	description := fmt.Sprintf("network with %d nodes running %s", numNodes, consensusAlgo)

	leaderKeyPair := testKeys.Ed25519KeyPairForTests(0)

	federationNodes := make(map[string]config.FederationNode)
	gossipPeers := make(map[string]config.GossipPeer)
	for i := 0; i < int(numNodes); i++ {
		publicKey := testKeys.Ed25519KeyPairForTests(i).PublicKey()
		federationNodes[publicKey.KeyForMap()] = config.NewHardCodedFederationNode(publicKey)
		gossipPeers[publicKey.KeyForMap()] = config.NewHardCodedGossipPeer(0, "")
	}

	sharedTamperingTransport := testGossipAdapter.NewTamperingTransport(testLogger, gossipAdapter.NewMemoryTransport(ctx, testLogger, federationNodes))

	network := &acceptanceNetwork{
		Network:            inmemory.NewNetwork(testLogger, sharedTamperingTransport),
		tamperingTransport: sharedTamperingTransport,
		description:        description,
	}

	for i := 0; i < numNodes; i++ {
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

		network.AddNode(keyPair, cfg, nativeProcessorAdapter.NewFakeCompiler())
	}

	return network

	// must call network.Start(ctx) to actually start the nodes in the network
}

type acceptanceNetwork struct {
	inmemory.Network

	tamperingTransport testGossipAdapter.Tamperer
	description        string
}

func (n *acceptanceNetwork) Start(ctx context.Context) {
	n.CreateAndStartNodes(ctx) // needs to start first so that nodes can register their listeners to it
}

func (n *acceptanceNetwork) WaitForTransactionInNodeState(ctx context.Context, txhash primitives.Sha256, nodeIndex int) {
	n.Nodes[nodeIndex].WaitForTransactionInState(ctx, txhash)
}

func (n *acceptanceNetwork) Description() string {
	return n.description
}

func (n *acceptanceNetwork) TransportTamperer() testGossipAdapter.Tamperer {
	return n.tamperingTransport
}

func (n *acceptanceNetwork) BlockPersistence(nodeIndex int) blockStorageAdapter.InMemoryBlockPersistence {
	return n.GetBlockPersistence(nodeIndex)
}

func (n *acceptanceNetwork) StatePersistence(nodeIndex int) stateStorageAdapter.TamperingStatePersistence {
	return n.GetStatePersistence(nodeIndex)
}

func (n *acceptanceNetwork) GetBenchmarkTokenContract() contracts.BenchmarkTokenClient {
	return contracts.NewContractClient(n)
}

func (n *acceptanceNetwork) DumpState() {
	for i := range n.Nodes {
		n.Logger.Info("state dump", log.Int("node", i), log.String("data", n.GetStatePersistence(i).Dump()))
	}
}

func (n *acceptanceNetwork) MockContract(fakeContractInfo *sdk.ContractInfo, code string) {

	// if needed, provide a fake implementation of this contract to all nodes
	for _, node := range n.Nodes {
		if fakeCompiler, ok := node.GetCompiler().(nativeProcessorAdapter.FakeCompiler); ok {
			fakeCompiler.ProvideFakeContract(fakeContractInfo, code)
		}
	}
}




package gamma

import (
	"context"
	"github.com/orbs-network/orbs-network-go/bootstrap/inmemory"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	blockPersistenceAdapter "github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	ethereumAdapter "github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	gossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	statePersistenceAdapter "github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
)

func NewDevelopmentNetwork(ctx context.Context, logger log.BasicLogger, metricRegistry metric.Registry) *inmemory.Network {
	numNodes := 2
	logger.Info("creating development network")

	federationNodes := map[string]config.FederationNode{}
	privateKeys := map[string]primitives.EcdsaSecp256K1PrivateKey{}

	var nodeOrder []primitives.NodeAddress
	for i := 0; i < int(numNodes); i++ {
		nodeAddress := keys.EcdsaSecp256K1KeyPairForTests(i).NodeAddress()
		federationNodes[nodeAddress.KeyForMap()] = config.NewHardCodedFederationNode(nodeAddress)
		privateKeys[nodeAddress.KeyForMap()] = keys.EcdsaSecp256K1KeyPairForTests(i).PrivateKey()
		nodeOrder = append(nodeOrder, nodeAddress)
	}
	sharedTransport := gossipAdapter.NewMemoryTransport(ctx, logger, federationNodes)
	cfgTemplate := config.TemplateForGamma(
		federationNodes,
		keys.EcdsaSecp256K1KeyPairForTests(0).NodeAddress(),
		consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS,
	)

	provider := func(idx int, nodeConfig config.NodeConfig, logger log.BasicLogger) (nativeProcessorAdapter.Compiler, ethereumAdapter.EthereumConnection, metric.Registry, blockPersistenceAdapter.BlockPersistence, statePersistenceAdapter.StatePersistence) {
		blockPersistence := blockPersistenceAdapter.NewTamperingInMemoryBlockPersistence(logger, nil, metricRegistry)
		compiler := nativeProcessorAdapter.NewNativeCompiler(cfgTemplate, logger)
		connection := ethereumAdapter.NewEthereumRpcConnection(cfgTemplate, logger)
		state := statePersistenceAdapter.NewInMemoryStatePersistence(metricRegistry)
		return compiler, connection, metricRegistry, blockPersistence, state
	}
	network := inmemory.NewNetworkWithNumOfNodes(federationNodes, nodeOrder, privateKeys, logger, cfgTemplate, sharedTransport, provider)
	network.CreateAndStartNodes(ctx, numNodes)
	return network
}

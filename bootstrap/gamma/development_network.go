package gamma

import (
	"context"
	"github.com/orbs-network/orbs-network-go/bootstrap/inmemory"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	gossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
)

func NewDevelopmentNetwork(ctx context.Context, logger log.BasicLogger) *inmemory.Network {
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

	network := inmemory.NewNetworkWithNumOfNodes(federationNodes, nodeOrder, privateKeys, logger, cfgTemplate, sharedTransport, nil)
	network.CreateAndStartNodes(ctx, numNodes)
	return network
}

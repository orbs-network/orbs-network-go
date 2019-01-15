package gamma

import (
	"context"
	"github.com/orbs-network/orbs-network-go/bootstrap/inmemory"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	gossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
)

func NewDevelopmentNetwork(ctx context.Context, logger log.BasicLogger, metricRegistry metric.Registry) *inmemory.Network {
	numNodes := 2
	logger.Info("creating development network")

	federationNodes := make(map[string]config.FederationNode)
	for i := 0; i < int(numNodes); i++ {
		nodeAddress := keys.EcdsaSecp256K1KeyPairForTests(i).NodeAddress()
		federationNodes[nodeAddress.KeyForMap()] = config.NewHardCodedFederationNode(nodeAddress)
	}
	sharedTransport := gossipAdapter.NewMemoryTransport(ctx, logger, federationNodes)
	cfgTemplate := config.TemplateForGamma(
		federationNodes,
		keys.EcdsaSecp256K1KeyPairForTests(0).NodeAddress(),
		consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS,
	)
	return inmemory.NewNetworkWithNumOfNodes(ctx, federationNodes, logger, cfgTemplate, metricRegistry, sharedTransport)
}

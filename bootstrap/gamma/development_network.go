package gamma

import (
	"context"
	"github.com/orbs-network/orbs-network-go/bootstrap/inmemory"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	gossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter/memory"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

func NewDevelopmentNetwork(ctx context.Context, logger log.BasicLogger, overrideConfigJson string) *inmemory.Network {
	numNodes := 2
	logger.Info("creating development network")

	validatorNodes := map[string]config.ValidatorNode{}
	privateKeys := map[string]primitives.EcdsaSecp256K1PrivateKey{}

	var nodeOrder []primitives.NodeAddress
	for i := 0; i < int(numNodes); i++ {
		nodeAddress := keys.EcdsaSecp256K1KeyPairForTests(i).NodeAddress()
		validatorNodes[nodeAddress.KeyForMap()] = config.NewHardCodedValidatorNode(nodeAddress)
		privateKeys[nodeAddress.KeyForMap()] = keys.EcdsaSecp256K1KeyPairForTests(i).PrivateKey()
		nodeOrder = append(nodeOrder, nodeAddress)
	}
	sharedTransport := gossipAdapter.NewTransport(ctx, logger, validatorNodes)
	cfgTemplate := config.TemplateForGamma(
		validatorNodes,
		keys.EcdsaSecp256K1KeyPairForTests(0).NodeAddress(),
	)

	if overrideConfigJson == "" {
		overrideConfigJson = "{}"
	}

	configWithOverrides, err := cfgTemplate.MergeWithFileConfig(overrideConfigJson)
	if err != nil {
		panic(err)
	}

	network := inmemory.NewNetworkWithNumOfNodes(validatorNodes, nodeOrder, privateKeys, logger, configWithOverrides, sharedTransport, nil)
	network.CreateAndStartNodes(ctx, numNodes)
	return network
}

// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package gamma

import (
	"context"
	"github.com/orbs-network/orbs-network-go/bootstrap/inmemory"
	"github.com/orbs-network/orbs-network-go/config"
	gossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter/memory"
	"github.com/orbs-network/orbs-network-go/services/transactionpool/adapter"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
)

func createGammaConfig(cfg ServerConfig, validatorNodes map[string]config.ValidatorNode) config.OverridableConfig{
	cfgTemplate := config.TemplateForGamma(
		validatorNodes, // TODO V2 get rid of this
		keys.EcdsaSecp256K1KeyPairForTests(0).NodeAddress(),
		cfg.ServerAddress,
		cfg.Profiling,
	)

	overrideConfigJson := "{}"
	if cfg.OverrideConfigJson != "" {
		overrideConfigJson = cfg.OverrideConfigJson
	}

	configWithOverrides, err := cfgTemplate.MergeWithFileConfig(overrideConfigJson)
	if err != nil {
		panic(err)
	}

	return configWithOverrides
}

func NewDevelopmentNetwork(ctx context.Context, logger log.Logger, maybeClock adapter.Clock, serverConfig ServerConfig) (*inmemory.Network, config.OverridableConfig) {
	numNodes := 4 // Comfortable number for LeanHelix if we choose to use it
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

	cfg := createGammaConfig(serverConfig, validatorNodes)

	network := inmemory.NewNetworkWithNumOfNodes(validatorNodes, nodeOrder, privateKeys, logger, cfg, sharedTransport, maybeClock, nil)
	network.CreateAndStartNodes(ctx, numNodes)
	return network, cfg
}

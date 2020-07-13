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
	managementAdapter "github.com/orbs-network/orbs-network-go/services/management/adapter"
	"github.com/orbs-network/orbs-network-go/services/transactionpool/adapter"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
)

func NewDevelopmentNetwork(ctx context.Context, logger log.Logger, maybeClock adapter.Clock, serverConfig ServerConfig) (*inmemory.Network, config.NodeConfig) {
	numNodes := 4 // Comfortable number for LeanHelix if we choose to use it
	logger.Info("creating development network")

	nodeOrder := keys.NodeAddressesForTests()[:numNodes]
	var nodeConfigs []config.NodeConfig
	for i, nodeAddress := range nodeOrder {
		cfg := config.TemplateForGamma(
			nodeAddress,
			keys.EcdsaSecp256K1KeyPairForTests(i).PrivateKey(),
			keys.EcdsaSecp256K1KeyPairForTests(0).NodeAddress(),
			serverConfig.ServerAddress,
			serverConfig.Profiling,
			serverConfig.OverrideConfigJson,
		)
		if cfg == nil {
			error := errors.Errorf("could not create gamma config with override string '%s'", serverConfig.OverrideConfigJson)
			logger.Error("cannot start", log.Error(error))
			panic(error)
		}
		nodeConfigs = append(nodeConfigs, cfg)
	}
	sharedTransport := gossipAdapter.NewTransport(ctx, logger, nodeOrder)
	sharedManagementProvider := managementAdapter.NewMemoryProvider(nodeOrder, nil /* with memory transport we don't need topology */, logger)

	network := inmemory.NewNetworkWithNumOfNodes(nodeOrder, nodeConfigs, logger, sharedTransport, sharedManagementProvider, maybeClock, nil)
	network.CreateAndStartNodes(ctx, numNodes)
	return network, nodeConfigs[0]
}

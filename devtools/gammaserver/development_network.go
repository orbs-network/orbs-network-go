package gammaserver

import (
	"context"
	"github.com/orbs-network/orbs-network-go/bootstrap/inmemory"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	gossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
)

func NewDevelopmentNetwork(ctx context.Context, logger log.BasicLogger) inmemory.NetworkDriver {
	numNodes := 2
	consensusAlgo := consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS
	logger.Info("creating development network")

	leaderKeyPair := keys.Ed25519KeyPairForTests(0)

	federationNodes := make(map[string]config.FederationNode)
	for i := 0; i < int(numNodes); i++ {
		publicKey := keys.Ed25519KeyPairForTests(i).PublicKey()
		federationNodes[publicKey.KeyForMap()] = config.NewHardCodedFederationNode(publicKey)
	}

	sharedTransport := gossipAdapter.NewMemoryTransport(ctx, logger, federationNodes)

	network := &inmemory.Network{
		Logger:    logger,
		Transport: sharedTransport,
	}

	for i := 0; i < numNodes; i++ {
		keyPair := keys.Ed25519KeyPairForTests(i)
		cfg := config.ForGamma(
			federationNodes,
			keyPair.PublicKey(),
			keyPair.PrivateKey(),
			leaderKeyPair.PublicKey(),
			consensusAlgo,
		)

		network.AddNode(keyPair, cfg, nativeProcessorAdapter.NewNativeCompiler(cfg, logger))
	}

	network.CreateAndStartNodes(ctx, numNodes) // must call network.Start(ctx) to actually start the nodes in the network

	return network
}

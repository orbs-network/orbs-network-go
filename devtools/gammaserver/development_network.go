package gammaserver

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/inprocess"
	gossipAdapter "github.com/orbs-network/orbs-network-go/inprocess/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
)

func NewDevelopmentNetwork(ctx context.Context, logger log.BasicLogger) inprocess.NetworkDriver {
	numNodes := 2
	consensusAlgo := consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS
	logger.Info("creating development network")

	leaderKeyPair := keys.Ed25519KeyPairForTests(0)

	federationNodes := make(map[string]config.FederationNode)
	gossipPeers := make(map[string]config.GossipPeer)
	for i := 0; i < int(numNodes); i++ {
		publicKey := keys.Ed25519KeyPairForTests(i).PublicKey()
		federationNodes[publicKey.KeyForMap()] = config.NewHardCodedFederationNode(publicKey)
		gossipPeers[publicKey.KeyForMap()] = config.NewHardCodedGossipPeer(0, "")
	}

	sharedTransport := gossipAdapter.NewChannelTransport(ctx, logger, federationNodes)

	nodes := make([]*inprocess.Node, numNodes)
	for i := range nodes {
		nodeKeyPair := keys.Ed25519KeyPairForTests(i)
		cfg := config.ForGamma(
			federationNodes,
			gossipPeers,
			nodeKeyPair.PublicKey(),
			nodeKeyPair.PrivateKey(),
			leaderKeyPair.PublicKey(),
			consensusAlgo,
		)
		compiler := nativeProcessorAdapter.NewNativeCompiler(cfg, logger)

		nodes[i] = inprocess.NewNode(i, nodeKeyPair, cfg, compiler)
	}

	network := &inprocess.Network{
		Nodes:     nodes,
		Logger:    logger,
		Transport: sharedTransport,
	}

	network.CreateAndStartNodes(ctx) // must call network.Start(ctx) to actually start the nodes in the network

	return network
}

// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package e2e

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/scribe/log"
	"os"
	"time"
)

var OwnerOfAllSupply = keys.Ed25519KeyPairForTests(5) // needs to be a constant across all e2e tests since we deploy the contract only once

// LOCAL_NETWORK_SIZE must remain identical to number of configured nodes in docker/test/benchmark-config
// Also Lean Helix consensus algo requires it to be >= 4 or it will panic
const LOCAL_NETWORK_SIZE = 4

type inProcessE2ENetwork struct {
	govnr.TreeSupervisor
	nodes          []*bootstrap.Node
	virtualChainId primitives.VirtualChainId
}

func NewInProcessE2EMgmtNetwork(virtualChainId primitives.VirtualChainId) *inProcessE2ENetwork {
	cleanNativeProcessorCache(virtualChainId)
	cleanBlockStorage(virtualChainId)

	return bootstrapE2ENetwork(LOCAL_NETWORK_SIZE, "mgmt", virtualChainId, false)
}

func NewInProcessE2EAppNetwork(virtualChainId primitives.VirtualChainId) *inProcessE2ENetwork {
	cleanNativeProcessorCache(virtualChainId)
	cleanBlockStorage(virtualChainId)

	return bootstrapE2ENetwork(0, "app", virtualChainId, true)
}

func (h *inProcessE2ENetwork) GracefulShutdownAndWipeDisk() {
	for _, node := range h.nodes {
		node.GracefulShutdown(context.TODO())
	}

	cleanNativeProcessorCache(h.virtualChainId)
	cleanBlockStorage(h.virtualChainId)
}

func bootstrapE2ENetwork(portOffset int, logFilePrefix string, virtualChainId primitives.VirtualChainId, deployBlocksFile bool) *inProcessE2ENetwork {
	net := &inProcessE2ENetwork{
		virtualChainId: virtualChainId,
	}
	gossipPortByNodeIndex := []int{}
	genesisValidatorNodes := make(map[string]config.ValidatorNode)
	gossipPeers := make(map[string]config.GossipPeer)

	for i := 0; i < LOCAL_NETWORK_SIZE; i++ {
		gossipPortByNodeIndex = append(gossipPortByNodeIndex, test.RandomPort())
		nodeAddress := keys.EcdsaSecp256K1KeyPairForTests(i).NodeAddress()
		genesisValidatorNodes[nodeAddress.KeyForMap()] = config.NewHardCodedValidatorNode(nodeAddress)
		gossipPeers[nodeAddress.KeyForMap()] = config.NewHardCodedGossipPeer(gossipPortByNodeIndex[i], "127.0.0.1", hex.EncodeToString(nodeAddress))
	}

	ethereumEndpoint := os.Getenv("ETHEREUM_ENDPOINT") //TODO v1 unite how this config is fetched

	_ = os.MkdirAll(config.GetProjectSourceRootPath()+"/_logs", 0755)
	console := log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter())

	logger := log.GetLogger().WithTags(
		log.String("_test", "e2e"),
		log.String("_branch", os.Getenv("GIT_BRANCH")),
		log.String("_commit", os.Getenv("GIT_COMMIT"))).
		WithOutput(console)
	leaderKeyPair := keys.EcdsaSecp256K1KeyPairForTests(0)
	for i := 0; i < LOCAL_NETWORK_SIZE; i++ {
		nodeKeyPair := keys.EcdsaSecp256K1KeyPairForTests(i)

		logFile, err := os.OpenFile(
			fmt.Sprintf("%s/_logs/%s-node%d-%v.log", config.GetProjectSourceRootPath(), logFilePrefix, i+1, time.Now().Format(time.RFC3339Nano)),
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0644)
		if err != nil {
			panic(err)
		}

		nodeLogger := logger.WithOutput(console, log.NewFormattingOutput(logFile, log.NewJsonFormatter()))
		processorArtifactPath := setUpProcessorArtifactPath(virtualChainId)

		cfg := config.
			ForE2E(
				fmt.Sprintf(":%d", START_HTTP_PORT+i+portOffset),
				virtualChainId,
				gossipPortByNodeIndex[i],
				nodeKeyPair.NodeAddress(),
				nodeKeyPair.PrivateKey(),
				gossipPeers,
				genesisValidatorNodes,
				getVirtualChainDataDir(virtualChainId),
				processorArtifactPath,
				ethereumEndpoint,
				leaderKeyPair.NodeAddress(),
				consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS,
			)

		if deployBlocksFile {
			deployBlockStorageFiles(cfg.BlockStorageFileSystemDataDir(), logger)
		}

		node := bootstrap.NewNode(cfg, nodeLogger)
		net.Supervise(node)
		net.nodes = append(net.nodes, node)
	}

	return net
}

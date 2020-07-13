// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package e2e

import (
	"context"
	"fmt"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/scribe/log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type inProcessE2ENetwork struct {
	govnr.TreeSupervisor
	nodes          []*bootstrap.Node
	virtualChainId primitives.VirtualChainId
	sync.RWMutex
	cfg config.NodeConfig
}

var OwnerOfAllSupply = keys.Ed25519KeyPairForTests(5) // needs to be a constant across all e2e tests since we deploy the contract only once

// LOCAL_NETWORK_SIZE must remain identical to number of configured nodes in docker/test/benchmark-config
// Also Lean Helix consensus algo requires it to be >= 4 or it will panic
const LOCAL_NETWORK_SIZE = 4

func NewInProcessE2EAppNetwork(virtualChainId primitives.VirtualChainId, randomer *loggerRandomer, experimentalExternalProcessorPluginPath string) *inProcessE2ENetwork {
	randomer.logger.Info("starting application network")
	cleanNativeProcessorCache(virtualChainId)
	cleanBlockStorage(virtualChainId)

	return bootstrapE2ENetwork("app", virtualChainId, true, randomer, experimentalExternalProcessorPluginPath)
}

func (h *inProcessE2ENetwork) GracefulShutdownAndWipeDisk() {
	for _, node := range h.nodes {
		node.GracefulShutdown(context.TODO())
	}

	cleanNativeProcessorCache(h.virtualChainId)
	cleanBlockStorage(h.virtualChainId)
}

func bootstrapE2ENetwork(logFilePrefix string, virtualChainId primitives.VirtualChainId, deployBlocksFile bool, tl *loggerRandomer, experimentalExternalProcessorPluginPath string) *inProcessE2ENetwork {
	net := &inProcessE2ENetwork{
		virtualChainId: virtualChainId,
	}

	ethereumEndpoint := os.Getenv("ETHEREUM_ENDPOINT") //TODO v1 unite how this config is fetched

	_ = os.MkdirAll(config.GetProjectSourceRootPath()+"/_logs", 0755)

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

		nodeLogger := tl.logger.WithOutput(tl.console, log.NewFormattingOutput(logFile, log.NewJsonFormatter()))
		processorArtifactPath, _ := getProcessorArtifactPath(virtualChainId)

		net.cfg = config.
			ForE2E(
				fmt.Sprintf(":%d", START_HTTP_PORT+i),
				virtualChainId,
				START_GOSSIP_PORT+i, // must match mgmt.json file
				nodeKeyPair.NodeAddress(),
				nodeKeyPair.PrivateKey(),
				filepath.Join(config.GetCurrentSourceFileDirPath(), "_data", "mgmt.json"),
				getVirtualChainDataDir(virtualChainId),
				processorArtifactPath,
				ethereumEndpoint,
				leaderKeyPair.NodeAddress(),
				consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX,
				experimentalExternalProcessorPluginPath,
			)

		// workaround for checking block syn in e2e - require the metrics node to sync his blocks from his peers
		// TODO: after topology mock control in e2e harness this should move
		if deployBlocksFile && i != METRICS_NODE_INDEX{
			deployBlockStorageFiles(net.cfg.BlockStorageFileSystemDataDir(), tl.logger)
		}

		node := bootstrap.NewNode(net.cfg, nodeLogger)
		net.Supervise(node)
		net.nodes = append(net.nodes, node)
	}

	return net
}

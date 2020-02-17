// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package bootstrap

import (
	"context"
	"fmt"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/bootstrap/httpserver"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter/filesystem"
	ethereumAdapter "github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	topologyProviderAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	topologyProviderFileAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter/file"
	topologyProviderMemoryAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter/memory"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/tcp"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/services/statestorage/adapter/memory"
	txPoolAdapter "github.com/orbs-network/orbs-network-go/services/transactionpool/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/scribe/log"
)

type Node struct {
	govnr.TreeSupervisor
	logic            NodeLogic
	cancelFunc       context.CancelFunc
	httpServer       *httpserver.HttpServer
	transport        *tcp.DirectTransport
	logger           log.Logger
	blockPersistence *filesystem.BlockPersistence
}

func getMetricRegistry(nodeConfig config.NodeConfig) metric.Registry {
	metricRegistry := metric.NewRegistry().WithVirtualChainId(nodeConfig.VirtualChainId()).WithNodeAddress(nodeConfig.NodeAddress())

	return metricRegistry
}

func NewNode(nodeConfig config.NodeConfig, logger log.Logger) *Node {
	ctx, ctxCancel := context.WithCancel(context.Background())
	config.NewValidator(logger).ValidateMainNode(nodeConfig) // this will panic if config does not pass validation

	nodeLogger := logger.WithTags(
		log.Node(nodeConfig.NodeAddress().String()),
		logfields.VirtualChainId(nodeConfig.VirtualChainId()),
	)
	metricRegistry := getMetricRegistry(nodeConfig)

	httpServer := httpserver.NewHttpServer(nodeConfig, nodeLogger, metricRegistry)

	blockPersistence, err := filesystem.NewBlockPersistence(nodeConfig, nodeLogger, metricRegistry)
	if err != nil {
		panic(fmt.Sprintf("failed initializing blocks database, err=%s", err.Error()))
	}

	var topologyProvider topologyProviderAdapter.TopologyProvider
	if len(nodeConfig.GossipTopologyFilePath()) == 0 {
		topologyProvider = topologyProviderMemoryAdapter.NewTopologyProvider(nodeConfig, nodeLogger)
	} else {
		topologyProviderFile := topologyProviderFileAdapter.NewTopologyProvider(nodeConfig, nodeLogger)
		err := topologyProviderFile.Update(ctx)
		if err != nil {
			panic(fmt.Sprintf("failed initializing topology from file, err=%s", err.Error()))
		}
		topologyProvider = topologyProviderFile
	}
	transport := tcp.NewDirectTransport(ctx, topologyProvider, nodeConfig, nodeLogger, metricRegistry)
	statePersistence := stateStorageAdapter.NewStatePersistence(metricRegistry)
	ethereumConnection := ethereumAdapter.NewEthereumRpcConnection(nodeConfig, logger, metricRegistry)
	nativeCompiler := nativeProcessorAdapter.NewNativeCompiler(nodeConfig, nodeLogger, metricRegistry)
	nodeLogic := NewNodeLogic(ctx, transport, blockPersistence, statePersistence, nil, nil, txPoolAdapter.NewSystemClock(), nativeCompiler, nodeLogger, metricRegistry, nodeConfig, ethereumConnection)

	httpServer.RegisterPublicApi(nodeLogic.PublicApi())

	n := &Node{
		logger:           nodeLogger,
		cancelFunc:       ctxCancel,
		logic:            nodeLogic,
		transport:        transport,
		httpServer:       httpServer,
		blockPersistence: blockPersistence,
	}

	ethereumConnection.ReportConnectionStatus(ctx)

	n.Supervise(ethereumConnection)
	n.Supervise(nodeLogic)
	n.Supervise(transport)
	n.Supervise(httpServer)
	return n
}

func (n *Node) GracefulShutdown(shutdownContext context.Context) {
	n.logger.Info("Shutting down")
	n.cancelFunc()
	supervised.ShutdownAllGracefully(shutdownContext, n.httpServer, n.transport, n.blockPersistence)
}

// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package bootstrap

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap/httpserver"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter/filesystem"
	ethereumAdapter "github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/tcp"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/services/statestorage/adapter/memory"
	txPoolAdapter "github.com/orbs-network/orbs-network-go/services/transactionpool/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/scribe/log"
	"time"
)

type Node interface {
	GracefulShutdown(shutdownContext context.Context)
	WaitUntilShutdown()
}

type node struct {
	logic         NodeLogic
	cancelFunc    context.CancelFunc
	httpServer    *httpserver.HttpServer
	doneReporting chan struct{}
	transport     *tcp.DirectTransport
}

func getMetricRegistry(nodeConfig config.NodeConfig) metric.Registry {
	metricRegistry := metric.NewRegistry().WithVirtualChainId(nodeConfig.VirtualChainId()).WithNodeAddress(nodeConfig.NodeAddress())

	return metricRegistry
}

func NewNode(nodeConfig config.NodeConfig, logger log.Logger) Node {
	ctx, ctxCancel := context.WithCancel(context.Background())
	config.NewValidator(logger).ValidateMainNode(nodeConfig) // this will panic if config does not pass validation

	nodeLogger := logger.WithTags(log.Node(nodeConfig.NodeAddress().String()))
	metricRegistry := getMetricRegistry(nodeConfig)

	blockPersistence, err := filesystem.NewBlockPersistence(ctx, nodeConfig, nodeLogger, metricRegistry)
	if err != nil {
		panic(fmt.Sprintf("failed initializing blocks database, err=%s", err.Error()))
	}

	transport := tcp.NewDirectTransport(ctx, nodeConfig, nodeLogger, metricRegistry)
	statePersistence := stateStorageAdapter.NewStatePersistence(metricRegistry)
	ethereumConnection := ethereumAdapter.NewEthereumRpcConnection(nodeConfig, logger)
	nativeCompiler := nativeProcessorAdapter.NewNativeCompiler(nodeConfig, nodeLogger, metricRegistry)
	nodeLogic := NewNodeLogic(ctx, transport, blockPersistence, statePersistence, nil, nil, txPoolAdapter.NewSystemClock(), nativeCompiler, nodeLogger, metricRegistry, nodeConfig, ethereumConnection)
	httpServer := httpserver.NewHttpServer(nodeConfig, nodeLogger, nodeLogic.PublicApi(), metricRegistry)

	return &node{
		cancelFunc:    ctxCancel,
		doneReporting: ethereumConnection.ReportConnectionStatus(ctx, metricRegistry, logger, 30*time.Second),
		logic:         nodeLogic,
		transport:     transport,
		httpServer:    httpServer,
	}
}

func (n node) GracefulShutdown(shutdownContext context.Context) {
	n.cancelFunc()
	synchronization.ShutdownAllGracefully(shutdownContext, n.httpServer, n.transport)
}

func (n node) WaitUntilShutdown() {
	synchronization.WaitForAllToShutdown(n.httpServer, n.logic, n.transport)
	<-n.doneReporting
}

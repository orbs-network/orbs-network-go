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
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter/filesystem"
	ethereumAdapter "github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/tcp"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/services/statestorage/adapter/memory"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Node interface {
	GracefulShutdown(timeout time.Duration)
	WaitUntilShutdown()
}

type node struct {
	httpServer   httpserver.HttpServer
	logic        NodeLogic
	shutdownCond *sync.Cond
	ctxCancel    context.CancelFunc
	logger       log.BasicLogger
}

func getMetricRegistry(nodeConfig config.NodeConfig) metric.Registry {
	metricRegistry := metric.NewRegistry()
	version := config.GetVersion()

	metricRegistry.NewText("Version.Semantic", version.Semantic)
	metricRegistry.NewText("Version.Commit", version.Commit)
	metricRegistry.NewText("Node.Address", nodeConfig.NodeAddress().String())

	return metricRegistry
}

func NewNode(nodeConfig config.NodeConfig, logger log.BasicLogger) Node {
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
	nativeCompiler := nativeProcessorAdapter.NewNativeCompiler(nodeConfig, nodeLogger)
	nodeLogic := NewNodeLogic(ctx, transport, blockPersistence, statePersistence, nil, nil, nativeCompiler, nodeLogger, metricRegistry, nodeConfig, ethereumConnection)
	httpServer := httpserver.NewHttpServer(nodeConfig, nodeLogger, nodeLogic.PublicApi(), metricRegistry)

	return &node{
		logic:        nodeLogic,
		httpServer:   httpServer,
		shutdownCond: sync.NewCond(&sync.Mutex{}),
		ctxCancel:    ctxCancel,
		logger:       nodeLogger,
	}
}

func (n *node) GracefulShutdown(timeout time.Duration) {
	n.ctxCancel()
	n.httpServer.GracefulShutdown(timeout)
	n.shutdownCond.Broadcast()
}

func (n *node) WaitUntilShutdown() {
	// if waiting for shutdown, listen for sigint and sigterm
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	supervised.GoOnce(n.logger, func() {
		<-signalChan
		n.logger.Info("terminating node gracefully due to os signal received")
		n.GracefulShutdown(0)
	})

	n.shutdownCond.L.Lock()
	n.shutdownCond.Wait()
	n.shutdownCond.L.Unlock()
}

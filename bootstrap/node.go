package bootstrap

import (
	"context"
	"github.com/orbs-network/orbs-network-go/bootstrap/httpserver"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	gossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	"sync"
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
}

func NewNode(nodeConfig config.NodeConfig, logger log.BasicLogger, httpAddress string) Node {
	ctx, ctxCancel := context.WithCancel(context.Background())

	nodeLogger := logger.WithTags(log.Node(nodeConfig.NodePublicKey().String()))
	metricRegistry := metric.NewRegistry()

	transport := gossipAdapter.NewDirectTransport(ctx, nodeConfig, nodeLogger)
	blockPersistence := blockStorageAdapter.NewInMemoryBlockPersistence(nodeLogger)
	statePersistence := stateStorageAdapter.NewInMemoryStatePersistence(metricRegistry)
	nativeCompiler := nativeProcessorAdapter.NewNativeCompiler(nodeConfig, nodeLogger)
	nodeLogic := NewNodeLogic(ctx, transport, blockPersistence, statePersistence, nativeCompiler, nodeLogger, metricRegistry, nodeConfig)
	httpServer := httpserver.NewHttpServer(httpAddress, nodeLogger, nodeLogic.PublicApi(), metricRegistry)

	return &node{
		logic:        nodeLogic,
		httpServer:   httpServer,
		shutdownCond: sync.NewCond(&sync.Mutex{}),
		ctxCancel:    ctxCancel,
	}
}

func (n *node) GracefulShutdown(timeout time.Duration) {
	n.ctxCancel()
	n.httpServer.GracefulShutdown(timeout)
	n.shutdownCond.Broadcast()
}

func (n *node) WaitUntilShutdown() {
	n.shutdownCond.L.Lock()
	n.shutdownCond.Wait()
	n.shutdownCond.L.Unlock()
}

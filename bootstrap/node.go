package bootstrap

import (
	"github.com/orbs-network/orbs-network-go/bootstrap/httpserver"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	gossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
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
}

func NewNode(
	httpAddress string,
	nodePublicKey primitives.Ed25519Pkey,
	transport gossipAdapter.Transport,
	isLeader bool,
	networkSize uint32,
) Node {

	nodeConfig := config.NewHardCodedConfig(networkSize, nodePublicKey)

	blockPersistence := blockStorageAdapter.NewLevelDbBlockPersistence(nodeConfig)
	stateStorageAdapter := stateStorageAdapter.NewLevelDbStatePersistence(nodeConfig)
	logger := instrumentation.NewStdoutLog()
	nodeLogic := NewNodeLogic(transport, blockPersistence, stateStorageAdapter, logger, nodeConfig, isLeader)
	httpServer := httpserver.NewHttpServer(httpAddress, logger, nodeLogic.PublicApi())

	return &node{
		logic:        nodeLogic,
		httpServer:   httpServer,
		shutdownCond: sync.NewCond(&sync.Mutex{}),
	}
}

func (n *node) GracefulShutdown(timeout time.Duration) {
	n.httpServer.GracefulShutdown(timeout)
	n.shutdownCond.Broadcast()
}

func (n *node) WaitUntilShutdown() {
	n.shutdownCond.L.Lock()
	n.shutdownCond.Wait()
	n.shutdownCond.L.Unlock()
}

package bootstrap

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap/httpserver"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	gossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"time"
)

type Node interface {
	GracefulShutdown(timeout time.Duration)
}

type node struct {
	httpServer httpserver.HttpServer
	logic      NodeLogic
}

func NewNode(
	address string,
	nodeId string,
	transport gossipAdapter.Transport,
	isLeader bool,
	networkSize uint32,
) Node {

	nodeConfig := config.NewHardCodedConfig(networkSize, nodeId)
	fmt.Println("Node config", nodeConfig)

	blockPersistence := blockStorageAdapter.NewLevelDbBlockPersistence(nodeConfig)
	stateStorageAdapter := stateStorageAdapter.NewLevelDbStatePersistence(nodeConfig)
	logger := instrumentation.NewStdoutLog()
	loopControl := instrumentation.NewSimpleLoop(logger)
	nodeLogic := NewNodeLogic(transport, blockPersistence, stateStorageAdapter, logger, loopControl, nodeConfig, isLeader)
	httpServer := httpserver.NewHttpServer(address, logger, nodeLogic.PublicApi())

	return &node{
		logic:      nodeLogic,
		httpServer: httpServer,
	}
}

func (n *node) GracefulShutdown(timeout time.Duration) {
	n.httpServer.GracefulShutdown(timeout)
}

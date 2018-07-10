package bootstrap

import (
	"fmt"
	"time"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/gossip"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/publicapi"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
)

type Node interface {
	GracefulShutdown(timeout time.Duration)
}

type node struct {
	httpServer publicapi.HttpServer
	logic      NodeLogic
}

func NewNode(
	address string,
	nodeId string,
	transport gossip.Transport,
	isLeader bool,
	networkSize uint32,
) Node {
	
	nodeConfig := config.NewHardCodedConfig(networkSize, nodeId)
	fmt.Println("Node config", nodeConfig)
	storage := blockStorageAdapter.NewBlockPersistence(nodeConfig)
	logger := instrumentation.NewStdoutLog()
	lc := instrumentation.NewSimpleLoop(logger)
	logic := NewNodeLogic(transport, storage, logger, lc, nodeConfig, isLeader)
	httpServer := publicapi.NewHttpServer(address, logger, logic.GetPublicApi())
	return &node{
		logic:      logic,
		httpServer: httpServer,
	}
}

func (n *node) GracefulShutdown(timeout time.Duration) {
	n.httpServer.GracefulShutdown(timeout)
}
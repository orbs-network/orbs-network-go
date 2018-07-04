package bootstrap

import (
	"github.com/orbs-network/orbs-network-go/blockstorage"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/test/harness/gossip"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/publicapi"
	"time"
)


type Node interface {
	GracefulShutdown(timeout time.Duration)
}

type node struct {
	httpServer publicapi.HttpServer
	logic NodeLogic
}


func NewNode(address string, nodeId string, isLeader bool, networkSize uint32) Node {
	nodeConfig := config.NewHardCodedConfig(networkSize, nodeId)

	transport := gossip.NewPausableTransport()
	storage := blockstorage.NewInMemoryBlockPersistence(nodeConfig)
	logger := instrumentation.NewStdoutLog()
	lc := instrumentation.NewSimpleLoop(logger)

	logic := NewNodeLogic(transport, storage, logger, lc, nodeConfig, isLeader)

	httpServer := publicapi.NewHttpServer(address, logger, logic.GetPublicApi())

	return &node {
		logic: logic,
		httpServer: httpServer,
	}
}

func (n *node) GracefulShutdown(timeout time.Duration) {
	n.httpServer.GracefulShutdown(timeout)
}

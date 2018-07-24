package bootstrap

import (
	"context"
	"github.com/orbs-network/orbs-network-go/bootstrap/httpserver"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	gossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
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

func NewNode(
	httpAddress string,
	nodePublicKey primitives.Ed25519Pkey,
	networkSize uint32,
	constantConsensusLeader primitives.Ed25519Pkey,
	activeConsensusAlgo consensus.ConsensusAlgoType, // TODO: move all of the config from the ctor, it's a smell
	transport gossipAdapter.Transport,
) Node {

	ctx, ctxCancel := context.WithCancel(context.Background())
	nodeConfig := config.NewHardCodedConfig(networkSize, nodePublicKey, constantConsensusLeader, activeConsensusAlgo)

	blockPersistence := blockStorageAdapter.NewLevelDbBlockPersistence(nodeConfig)
	stateStorageAdapter := stateStorageAdapter.NewLevelDbStatePersistence(nodeConfig)
	logger := instrumentation.NewStdoutLog()
	nodeLogic := NewNodeLogic(ctx, transport, blockPersistence, stateStorageAdapter, logger, nodeConfig)
	httpServer := httpserver.NewHttpServer(httpAddress, logger, nodeLogic.PublicApi())

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

package bootstrap

import (
	"context"
	"github.com/orbs-network/orbs-network-go/bootstrap/httpserver"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	gossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
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
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	federationNodes map[string]config.FederationNode,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
	logger log.BasicLogger,
	transport gossipAdapter.Transport,
) Node {
	ctx, ctxCancel := context.WithCancel(context.Background())
	nodeConfig := config.ForProduction(
		federationNodes,
		nodePublicKey,
		nodePrivateKey,
		constantConsensusLeader,
		activeConsensusAlgo,
	)

	nodeLogger := logger.For(log.Node(nodePublicKey.String()))

	blockPersistence := blockStorageAdapter.NewInMemoryBlockPersistence()
	stateStorageAdapter := stateStorageAdapter.NewInMemoryStatePersistence()
	nodeLogic := NewNodeLogic(ctx, transport, blockPersistence, stateStorageAdapter, nodeLogger, nodeConfig)
	httpServer := httpserver.NewFastHttpServer(httpAddress, nodeLogger, nodeLogic.PublicApi())

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

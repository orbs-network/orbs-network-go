package inmemory

import (
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/services"

	blockStorageAdapter "github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	ethereumAdapter "github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
)

// Represents an in-memory Orbs node, that uses in-memory storage and communicates with its peers via in-memory gossip
// Useful for in-process tests and simulating Orbs chains during development
type Node struct {
	index                             int
	name                              string
	config                            config.NodeConfig
	BlockPersistence                  blockStorageAdapter.BlockPersistence
	StatePersistence                  stateStorageAdapter.StatePersistence
	StateBlockHeightTracker           *synchronization.BlockTracker
	transactionPoolBlockHeightTracker *synchronization.BlockTracker
	nativeCompiler                    nativeProcessorAdapter.Compiler
	ethereumConnection                ethereumAdapter.EthereumConnection
	nodeLogic                         bootstrap.NodeLogic
	metricRegistry                    metric.Registry
}

func (n *Node) GetPublicApi() services.PublicApi {
	return n.nodeLogic.PublicApi()
}

func (n *Node) Started() bool {
	return n.nodeLogic != nil
}

func (n *Node) Destroy() {
	n.nodeLogic = nil
}

func (n *Node) GetTransactionPoolBlockHeightTracker() *synchronization.BlockTracker {
	return n.transactionPoolBlockHeightTracker
}

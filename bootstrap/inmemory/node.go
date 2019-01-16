package inmemory

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"

	blockStorageAdapter "github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	ethereumAdapter "github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	harnessStateStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/statestorage/adapter"
)

// Represents an in-memory Orbs node, that uses in-memory storage and communicates with its peers via in-memory gossip
// Useful for in-process tests and simulating Orbs chains during development
type Node struct {
	index                             int
	name                              string
	config                            config.NodeConfig
	BlockPersistence                  blockStorageAdapter.TamperingInMemoryBlockPersistence
	StatePersistence                  harnessStateStorageAdapter.DumpingStatePersistence
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

func (n *Node) WaitForTransactionInState(ctx context.Context, txHash primitives.Sha256) primitives.BlockHeight {
	blockHeight := n.BlockPersistence.WaitForTransaction(ctx, txHash)
	err := n.StateBlockHeightTracker.WaitForBlock(ctx, blockHeight)
	if err != nil {
		test.DebugPrintGoroutineStacks() // since test timed out, help find deadlocked goroutines
		panic(fmt.Sprintf("statePersistence.WaitUntilCommittedBlockOfHeight failed: %s", err.Error()))
	}
	return blockHeight
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

package harness

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/harness/contracts"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type InProcessNetwork interface {
	PublicApi(nodeIndex int) services.PublicApi
	GetCounterContract() contracts.CounterClient
}

type inProcessNetwork struct {
	nodes  []*networkNode
	logger log.BasicLogger
	transport adapter.Transport
}

type networkNode struct {
	index            int
	name             string
	config           config.NodeConfig
	blockPersistence blockStorageAdapter.InMemoryBlockPersistence
	statePersistence stateStorageAdapter.TamperingStatePersistence
	nativeCompiler   nativeProcessorAdapter.Compiler
	nodeLogic        bootstrap.NodeLogic
	metricRegistry   metric.Registry
}

func (n *inProcessNetwork) createAndStartNodes(ctx context.Context) {
	for _, node := range n.nodes {
		node.nodeLogic = bootstrap.NewNodeLogic(
			ctx,
			n.transport,
			node.blockPersistence,
			node.statePersistence,
			node.nativeCompiler,
			n.logger.WithTags(log.Node(node.name)),
			node.metricRegistry,
			node.config,
		)
	}
}

func (n *networkNode) GetPublicApi() services.PublicApi {
	return n.nodeLogic.PublicApi()
}

func (n *networkNode) GetCompiler() nativeProcessorAdapter.Compiler {
	return n.nativeCompiler
}

func (n *networkNode) WaitForTransactionInState(ctx context.Context, txhash primitives.Sha256) {
	blockHeight := n.blockPersistence.WaitForTransaction(ctx, txhash)
	err := n.statePersistence.WaitUntilCommittedBlockOfHeight(ctx, blockHeight)
	if err != nil {
		test.DebugPrintGoroutineStacks() // since test timed out, help find deadlocked goroutines
		panic(fmt.Sprintf("statePersistence.WaitUntilCommittedBlockOfHeight failed: %s", err.Error()))
	}
}

func (n *inProcessNetwork) PublicApi(nodeIndex int) services.PublicApi {
	return n.nodes[nodeIndex].nodeLogic.PublicApi()
}

func (n *inProcessNetwork) nodesAsContractAPIProviders() []contracts.APIProvider {
	var apis []contracts.APIProvider
	for _, node := range n.nodes {
		apis = append(apis, node)
	}
	return apis
}

func (n *inProcessNetwork) GetCounterContract() contracts.CounterClient {
	return contracts.NewContractClient(n.nodesAsContractAPIProviders(), n.logger)
}

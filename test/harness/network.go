package harness

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/harness/contracts"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	gossipAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type InProcessNetwork interface {
	PublicApi(nodeIndex int) services.PublicApi
	GetCounterContract() contracts.CounterClient
}

type InProcessTestNetwork interface {
	InProcessNetwork
	GetBenchmarkTokenContract() contracts.BenchmarkTokenClient
	GossipTransport() gossipAdapter.TamperingTransport
	Description() string
	BlockPersistence(nodeIndex int) blockStorageAdapter.InMemoryBlockPersistence
	DumpState()
	WaitForTransactionInState(ctx context.Context, nodeIndex int, txhash primitives.Sha256)
	Size() int
	MetricsString(nodeIndex int) string
}

type inProcessNetwork struct {
	nodes           []*networkNode
	gossipTransport gossipAdapter.TamperingTransport
	description     string
	testLogger      log.BasicLogger
}

func (n *inProcessNetwork) Start(ctx context.Context) InProcessNetwork {
	n.gossipTransport.Start(ctx) // needs to start first so that nodes can register their listeners to it

	for _, node := range n.nodes {
		node.nodeLogic = bootstrap.NewNodeLogic(
			ctx,
			n.gossipTransport,
			node.blockPersistence,
			node.statePersistence,
			node.nativeCompiler,
			n.testLogger.WithTags(log.Node(node.name)),
			node.metricRegistry,
			node.config,
		)
	}
	return n
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

func (n *inProcessNetwork) WaitForTransactionInState(ctx context.Context, nodeIndex int, txhash primitives.Sha256) {
	n.nodes[nodeIndex].WaitForTransactionInState(ctx, txhash)
}

func (n *inProcessNetwork) MetricsString(i int) string {
	return n.nodes[i].metricRegistry.String()
}

func (n *inProcessNetwork) Description() string {
	return n.description
}

func (n *inProcessNetwork) GossipTransport() gossipAdapter.TamperingTransport {
	return n.gossipTransport
}

func (n *inProcessNetwork) PublicApi(nodeIndex int) services.PublicApi {
	return n.nodes[nodeIndex].nodeLogic.PublicApi()
}

func (n *inProcessNetwork) BlockPersistence(nodeIndex int) blockStorageAdapter.InMemoryBlockPersistence {
	return n.nodes[nodeIndex].blockPersistence
}

func (n *inProcessNetwork) GetCounterContract() contracts.CounterClient {
	return contracts.NewContractClient(n.nodesAsContractAPIProviders(), n.testLogger)
}

func (n *inProcessNetwork) GetBenchmarkTokenContract() contracts.BenchmarkTokenClient {
	return contracts.NewContractClient(n.nodesAsContractAPIProviders(), n.testLogger)
}

func (n *inProcessNetwork) DumpState() {
	for i := range n.nodes {
		n.testLogger.Info("state dump", log.Int("node", i), log.String("data", n.nodes[i].statePersistence.Dump()))
	}
}

func (n *inProcessNetwork) Size() int {
	return len(n.nodes)
}

func (n *inProcessNetwork) nodesAsContractAPIProviders() []contracts.APIProvider {
	var apis []contracts.APIProvider
	for _, node := range n.nodes {
		apis = append(apis, node)
	}
	return apis
}

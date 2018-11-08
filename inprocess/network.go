package inprocess

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/keys"
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

type NetworkDriver interface {
	PublicApi(nodeIndex int) services.PublicApi
	GetCounterContract() contracts.CounterClient
}

type Network struct {
	Nodes     []*Node
	Logger    log.BasicLogger
	Transport adapter.Transport
}

type Node struct {
	index            int
	name             string
	config           config.NodeConfig
	blockPersistence blockStorageAdapter.InMemoryBlockPersistence
	statePersistence stateStorageAdapter.TamperingStatePersistence
	nativeCompiler   nativeProcessorAdapter.Compiler
	nodeLogic        bootstrap.NodeLogic
	metricRegistry   metric.Registry
}

func NewNode(i int, nodeKeyPair *keys.Ed25519KeyPair, cfg config.NodeConfig, compiler nativeProcessorAdapter.Compiler) *Node {
	node := &Node{}
	node.index = i
	node.name = fmt.Sprintf("%s", nodeKeyPair.PublicKey()[:3])
	node.config = cfg
	node.statePersistence = stateStorageAdapter.NewTamperingStatePersistence()
	node.blockPersistence = blockStorageAdapter.NewInMemoryBlockPersistence()
	node.nativeCompiler = compiler
	node.metricRegistry = metric.NewRegistry()
	return node
}

func (n *Network) CreateAndStartNodes(ctx context.Context) {
	for _, node := range n.Nodes {
		node.nodeLogic = bootstrap.NewNodeLogic(
			ctx,
			n.Transport,
			node.blockPersistence,
			node.statePersistence,
			node.nativeCompiler,
			n.Logger.WithTags(log.Node(node.name)),
			node.metricRegistry,
			node.config,
		)
	}
}

func (n *Node) GetPublicApi() services.PublicApi {
	return n.nodeLogic.PublicApi()
}

func (n *Node) GetCompiler() nativeProcessorAdapter.Compiler {
	return n.nativeCompiler
}

func (n *Node) WaitForTransactionInState(ctx context.Context, txhash primitives.Sha256) {
	blockHeight := n.blockPersistence.WaitForTransaction(ctx, txhash)
	err := n.statePersistence.WaitUntilCommittedBlockOfHeight(ctx, blockHeight)
	if err != nil {
		test.DebugPrintGoroutineStacks() // since test timed out, help find deadlocked goroutines
		panic(fmt.Sprintf("statePersistence.WaitUntilCommittedBlockOfHeight failed: %s", err.Error()))
	}
}

func (n *Network) PublicApi(nodeIndex int) services.PublicApi {
	return n.Nodes[nodeIndex].nodeLogic.PublicApi()
}

func (n *Network) GetAPIProviders() []contracts.APIProvider {
	var apis []contracts.APIProvider
	for _, node := range n.Nodes {
		apis = append(apis, node)
	}
	return apis
}

func (n *Network) GetCounterContract() contracts.CounterClient {
	return contracts.NewContractClient(n.GetAPIProviders(), n.Logger)
}

func (n *Network) GetBlockPersistence(nodeIndex int) blockStorageAdapter.InMemoryBlockPersistence {
	return n.Nodes[nodeIndex].blockPersistence
}

func (n *Network) GetStatePersistence(i int) stateStorageAdapter.TamperingStatePersistence {
	return n.Nodes[i].statePersistence
}

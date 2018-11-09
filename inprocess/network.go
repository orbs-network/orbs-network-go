package inprocess

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/keys"
	"github.com/orbs-network/orbs-network-go/inprocess/contracts"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type NetworkDriver interface {
	contracts.ContractAPI
	PublicApi(nodeIndex int) services.PublicApi
	GetCounterContract() contracts.CounterClient
	Size() int
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

func NewNetwork(logger log.BasicLogger, transport adapter.Transport) Network {
	return Network{Logger: logger, Transport: transport}
}

func (n *Network) AddNode(nodeKeyPair *keys.Ed25519KeyPair, cfg config.NodeConfig, compiler nativeProcessorAdapter.Compiler) {
	node := &Node{}
	node.index = len(n.Nodes)
	node.name = fmt.Sprintf("%s", nodeKeyPair.PublicKey()[:3])
	node.config = cfg
	node.statePersistence = stateStorageAdapter.NewTamperingStatePersistence()
	node.blockPersistence = blockStorageAdapter.NewInMemoryBlockPersistence()
	node.nativeCompiler = compiler
	node.metricRegistry = metric.NewRegistry()

	n.Nodes = append(n.Nodes, node)
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

func (n *Network) GetCounterContract() contracts.CounterClient {
	return contracts.NewContractClient(n, n.Logger)
}

func (n *Network) GetBlockPersistence(nodeIndex int) blockStorageAdapter.InMemoryBlockPersistence {
	return n.Nodes[nodeIndex].blockPersistence
}

func (n *Network) GetStatePersistence(i int) stateStorageAdapter.TamperingStatePersistence {
	return n.Nodes[i].statePersistence
}

func (n *Network) Size() int {
	return len(n.Nodes)
}

func (n *Network) SendTransaction(ctx context.Context, tx *protocol.SignedTransactionBuilder, nodeIndex int) chan *client.SendTransactionResponse {
	ch := make(chan *client.SendTransactionResponse)
	supervised.GoOnce(n.Logger, func() {
		publicApi := n.Nodes[nodeIndex].GetPublicApi()
		output, err := publicApi.SendTransaction(ctx, &services.SendTransactionInput{
			ClientRequest: (&client.SendTransactionRequestBuilder{SignedTransaction: tx}).Build(),
		})
		if err != nil {
			panic(fmt.Sprintf("error sending transaction: %v", err)) // TODO: improve
		}
		ch <- output.ClientResponse
	})
	return ch
}

func (n *Network) CallMethod(ctx context.Context, tx *protocol.TransactionBuilder, nodeIndex int) chan uint64 {

	ch := make(chan uint64)
	supervised.GoOnce(n.Logger, func() {
		publicApi := n.Nodes[nodeIndex].GetPublicApi()
		output, err := publicApi.CallMethod(ctx, &services.CallMethodInput{
			ClientRequest: (&client.CallMethodRequestBuilder{Transaction: tx}).Build(),
		})
		if err != nil {
			panic(fmt.Sprintf("error calling method: %v", err)) // TODO: improve
		}
		outputArgsIterator := builders.ClientCallMethodResponseOutputArgumentsDecode(output.ClientResponse)
		ch <- outputArgsIterator.NextArguments().Uint64Value()
	})
	return ch
}


func (n *Network) WaitForTransactionInState(ctx context.Context, txhash primitives.Sha256) {
	for _, node := range n.Nodes {
		node.WaitForTransactionInState(ctx, txhash)
	}
}


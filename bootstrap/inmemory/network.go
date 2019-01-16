package inmemory

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	ethereumAdapter "github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization"
	harnessStateStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"math"
)

// Represents an in-process network connecting a group of in-memory Nodes together using the provided Transport
type Network struct {
	Nodes     []*Node
	Logger    log.BasicLogger
	Transport adapter.Transport
}

type nodeDependencyProvider func(nodeConfig config.NodeConfig, logger log.BasicLogger) (nativeProcessorAdapter.Compiler, ethereumAdapter.EthereumConnection, metric.Registry, blockStorageAdapter.TamperingInMemoryBlockPersistence)

func NewNetworkWithNumOfNodes(
	federation map[string]config.FederationNode,
	privateKeys map[string]primitives.EcdsaSecp256K1PrivateKey,
	parent log.BasicLogger,
	cfgTemplate config.OverridableConfig,
	transport adapter.Transport,
	provider nodeDependencyProvider,
) *Network {

	network := &Network{
		Logger:    parent,
		Transport: transport,
	}

	for stringAddress, federationNode := range federation {

		cfg := cfgTemplate.ForNode(federationNode.NodeAddress(), privateKeys[stringAddress])

		nodeLogger := parent.WithTags(log.Node(cfg.NodeAddress().String()))
		compiler, ethereumConnection, metricRegistry, blockPersistence := provider(cfg, nodeLogger)

		network.addNode(fmt.Sprintf("%s", federationNode.NodeAddress()[:3]), cfg, blockPersistence, compiler, ethereumConnection, metricRegistry, nodeLogger)
	}

	return network // call network.CreateAndStartNodes to launch nodes in the network
}

func (n *Network) addNode(name string, cfg config.NodeConfig, blockPersistence blockStorageAdapter.TamperingInMemoryBlockPersistence, compiler nativeProcessorAdapter.Compiler, ethereumConnection ethereumAdapter.EthereumConnection, metricRegistry metric.Registry, logger log.BasicLogger) {

	node := &Node{}
	node.index = len(n.Nodes)
	node.name = name
	node.config = cfg
	node.StatePersistence = harnessStateStorageAdapter.NewDumpingStatePersistence(metricRegistry, logger)
	node.StateBlockHeightTracker = synchronization.NewBlockTracker(logger, 0, math.MaxUint16)
	node.transactionPoolBlockHeightTracker = synchronization.NewBlockTracker(logger, 0, math.MaxUint16)
	node.BlockPersistence = blockPersistence
	node.nativeCompiler = compiler
	node.ethereumConnection = ethereumConnection
	node.metricRegistry = metricRegistry

	n.Nodes = append(n.Nodes, node)
}

func (n *Network) CreateAndStartNodes(ctx context.Context, numOfNodesToStart int) {
	for i, node := range n.Nodes {
		if i >= numOfNodesToStart {
			return
		}

		node.nodeLogic = bootstrap.NewNodeLogic(
			ctx,
			n.Transport,
			node.BlockPersistence,
			node.StatePersistence,
			node.StateBlockHeightTracker,
			node.transactionPoolBlockHeightTracker,
			node.nativeCompiler,
			n.Logger.WithTags(log.Node(node.name)),
			node.metricRegistry,
			node.config,
			node.ethereumConnection,
		)
		defer node.transactionPoolBlockHeightTracker.WaitForBlock(ctx, 1)
	}
}

func (n *Network) PublicApi(nodeIndex int) services.PublicApi {
	return n.Nodes[nodeIndex].nodeLogic.PublicApi()
}

func (n *Network) SendTransaction(ctx context.Context, builder *protocol.SignedTransactionBuilder, nodeIndex int) (*client.SendTransactionResponse, primitives.Sha256) {
	n.assertStarted(nodeIndex)
	ch := make(chan *client.SendTransactionResponse)

	transactionRequestBuilder := &client.SendTransactionRequestBuilder{SignedTransaction: builder}
	txHash := digest.CalcTxHash(transactionRequestBuilder.SignedTransaction.Transaction.Build())

	go func() {
		defer close(ch)
		publicApi := n.Nodes[nodeIndex].GetPublicApi()
		output, err := publicApi.SendTransaction(ctx, &services.SendTransactionInput{
			ClientRequest: transactionRequestBuilder.Build(),
		})
		if output == nil {
			panic(fmt.Sprintf("error sending transaction: %v", err)) // TODO(https://github.com/orbs-network/orbs-network-go/issues/531): improve
		}

		select {
		case ch <- output.ClientResponse:
		case <-ctx.Done():
		}
	}()

	return <-ch, txHash
}

func (n *Network) SendTransactionInBackground(ctx context.Context, builder *protocol.SignedTransactionBuilder, nodeIndex int) {
	n.assertStarted(nodeIndex)

	go func() {
		publicApi := n.Nodes[nodeIndex].GetPublicApi()
		output, err := publicApi.SendTransaction(ctx, &services.SendTransactionInput{
			ClientRequest:     (&client.SendTransactionRequestBuilder{SignedTransaction: builder}).Build(),
			ReturnImmediately: 1,
		})
		if output == nil {
			panic(fmt.Sprintf("error sending transaction: %v", err)) // TODO(https://github.com/orbs-network/orbs-network-go/issues/531): improve
		}
	}()
}

func (n *Network) RunQuery(ctx context.Context, builder *protocol.SignedQueryBuilder, nodeIndex int) *client.RunQueryResponse {
	n.assertStarted(nodeIndex)

	ch := make(chan *client.RunQueryResponse)
	go func() {
		defer close(ch)
		publicApi := n.Nodes[nodeIndex].GetPublicApi()
		output, err := publicApi.RunQuery(ctx, &services.RunQueryInput{
			ClientRequest: (&client.RunQueryRequestBuilder{SignedQuery: builder}).Build(),
		})
		if output == nil {
			panic(fmt.Sprintf("error calling method: %v", err)) // TODO(https://github.com/orbs-network/orbs-network-go/issues/531): improve
		}
		select {
		case ch <- output.ClientResponse:
		case <-ctx.Done():
		}
	}()
	return <-ch
}

func (n *Network) assertStarted(nodeIndex int) {
	if !n.Nodes[nodeIndex].Started() {
		panic(fmt.Errorf("accessing a stopped node %d", nodeIndex))
	}
}

func (n *Network) WaitForTransactionInState(ctx context.Context, txHash primitives.Sha256) {
	for _, node := range n.Nodes {
		if node.Started() {
			h := node.WaitForTransactionInState(ctx, txHash)
			n.Logger.Info("WaitForTransactionInState found tx in state", log.BlockHeight(h), log.Node(node.name), log.Transaction(txHash))
		}
	}
}

func (n *Network) Destroy() {
	for _, node := range n.Nodes {
		node.Destroy()
	}
}

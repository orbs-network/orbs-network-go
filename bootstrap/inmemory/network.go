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
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter/memory"
	ethereumAdapter "github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"math"
	"sync"
)

// Represents an in-process network connecting a group of in-memory Nodes together using the provided Transport
type Network struct {
	Nodes     []*Node
	Logger    log.BasicLogger
	Transport adapter.Transport
}

type NodeDependencies struct {
	Compiler         nativeProcessorAdapter.Compiler
	EtherConnection  ethereumAdapter.EthereumConnection
	BlockPersistence blockStorageAdapter.BlockPersistence
	StatePersistence stateStorageAdapter.StatePersistence
}
type nodeDependencyProvider func(idx int, nodeConfig config.NodeConfig, logger log.BasicLogger, metricRegistry metric.Registry) *NodeDependencies

func NewNetworkWithNumOfNodes(
	federation map[string]config.FederationNode,
	nodeOrder []primitives.NodeAddress,
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
	parent.Info("acceptance network node order", log.StringableSlice("addresses", nodeOrder))

	for _, address := range nodeOrder {
		federationNode := federation[address.KeyForMap()]
		cfg := cfgTemplate.ForNode(address, privateKeys[address.KeyForMap()])
		metricRegistry := metric.NewRegistry()

		nodeLogger := parent.WithTags(log.Node(cfg.NodeAddress().String()))
		dep := &NodeDependencies{}
		if provider == nil {
			dep.BlockPersistence = memory.NewBlockPersistence(nodeLogger, metricRegistry)
			dep.Compiler = nativeProcessorAdapter.NewNativeCompiler(cfgTemplate, nodeLogger)
			dep.EtherConnection = ethereumAdapter.NewEthereumRpcConnection(cfgTemplate, nodeLogger)
			dep.StatePersistence = stateStorageAdapter.NewInMemoryStatePersistence(metricRegistry)
		} else {
			dep = provider(len(network.Nodes), cfg, nodeLogger, metricRegistry)
		}

		network.addNode(fmt.Sprintf("%s", federationNode.NodeAddress()[:3]), cfg, dep.BlockPersistence, dep.StatePersistence, dep.Compiler, dep.EtherConnection, metricRegistry, nodeLogger)
	}

	return network // call network.CreateAndStartNodes to launch nodes in the network
}

func (n *Network) addNode(name string, cfg config.NodeConfig, blockPersistence blockStorageAdapter.BlockPersistence, statePersistence stateStorageAdapter.StatePersistence, compiler nativeProcessorAdapter.Compiler, ethereumConnection ethereumAdapter.EthereumConnection, metricRegistry metric.Registry, logger log.BasicLogger) {

	node := &Node{}
	node.index = len(n.Nodes)
	node.name = name
	node.config = cfg
	node.StatePersistence = statePersistence
	node.StateBlockHeightTracker = synchronization.NewBlockTracker(logger, 0, math.MaxUint16)
	node.transactionPoolBlockHeightTracker = synchronization.NewBlockTracker(logger, 0, math.MaxUint16)
	node.BlockPersistence = blockPersistence
	node.nativeCompiler = compiler
	node.ethereumConnection = ethereumConnection
	node.metricRegistry = metricRegistry

	n.Nodes = append(n.Nodes, node)
}

func (n *Network) CreateAndStartNodes(ctx context.Context, numOfNodesToStart int) {
	wg := sync.WaitGroup{}
	for i, node := range n.Nodes {
		if i >= numOfNodesToStart {
			break
		}
		wg.Add(1)

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
		go func(nx *Node) { // nodes should not block each other from executing wait
			if err := nx.transactionPoolBlockHeightTracker.WaitForBlock(ctx, 1); err != nil {
				panic(fmt.Sprintf("node %v did not reach block 1", node.name))
			}
			wg.Done()
		}(node)
	}
	wg.Wait()
}

func (n *Network) PublicApi(nodeIndex int) services.PublicApi {
	return n.Nodes[nodeIndex].nodeLogic.PublicApi()
}

type sendTxResp struct {
	res *services.SendTransactionOutput
	err error
}

func (n *Network) SendTransaction(ctx context.Context, builder *protocol.SignedTransactionBuilder, nodeIndex int) (*client.SendTransactionResponse, primitives.Sha256) {
	n.assertStarted(nodeIndex)
	ch := make(chan sendTxResp)

	transactionRequestBuilder := &client.SendTransactionRequestBuilder{SignedTransaction: builder}
	txHash := digest.CalcTxHash(transactionRequestBuilder.SignedTransaction.Transaction.Build())

	go func() {
		defer close(ch)
		publicApi := n.Nodes[nodeIndex].GetPublicApi()
		output, err := publicApi.SendTransaction(ctx, &services.SendTransactionInput{
			ClientRequest: transactionRequestBuilder.Build(),
		})

		select {
		case ch <- sendTxResp{res: output, err: err}:
		case <-ctx.Done():
			ch <- sendTxResp{err: errors.Wrap(ctx.Err(), "aborted send tx")}
		}
	}()

	out := <-ch
	if out.res == nil {
		panic(fmt.Sprintf("error in send transaction: %v", out.err)) // TODO(https://github.com/orbs-network/orbs-network-go/issues/531): improve
	}
	return out.res.ClientResponse, txHash
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

type getTxStatResp struct {
	res *services.GetTransactionStatusOutput
	err error
}

func (n *Network) GetTransactionStatus(ctx context.Context, txHash primitives.Sha256, nodeIndex int) *client.GetTransactionStatusResponse {
	n.assertStarted(nodeIndex)

	ch := make(chan getTxStatResp)
	go func() {
		defer close(ch)
		publicApi := n.Nodes[nodeIndex].GetPublicApi()
		output, err := publicApi.GetTransactionStatus(ctx, &services.GetTransactionStatusInput{
			ClientRequest: (&client.GetTransactionStatusRequestBuilder{
				TransactionRef: builders.TransactionRef().WithTxHash(txHash).Builder(),
			}).Build(),
		})
		select {
		case ch <- getTxStatResp{res: output, err: err}:
		case <-ctx.Done():
			ch <- getTxStatResp{err: errors.Wrap(ctx.Err(), "aborted get tx status")}
		}
	}()
	out := <-ch
	if out.res == nil {
		panic(fmt.Sprintf("error in get tx status: %v", out.err)) // TODO(https://github.com/orbs-network/orbs-network-go/issues/531): improve
	}
	return out.res.ClientResponse
}

type runQueryResp struct {
	res *services.RunQueryOutput
	err error
}

func (n *Network) RunQuery(ctx context.Context, builder *protocol.SignedQueryBuilder, nodeIndex int) *client.RunQueryResponse {
	n.assertStarted(nodeIndex)

	ch := make(chan runQueryResp)
	go func() {
		defer close(ch)
		publicApi := n.Nodes[nodeIndex].GetPublicApi()
		output, err := publicApi.RunQuery(ctx, &services.RunQueryInput{
			ClientRequest: (&client.RunQueryRequestBuilder{SignedQuery: builder}).Build(),
		})

		select {
		case ch <- runQueryResp{res: output, err: err}:
		case <-ctx.Done():
			ch <- runQueryResp{err: errors.Wrap(ctx.Err(), "aborted run query")}
		}
	}()
	out := <-ch
	if out.res == nil {
		panic(fmt.Sprintf("error in run query: %v", out.err)) // TODO(https://github.com/orbs-network/orbs-network-go/issues/531): improve
	}
	return out.res.ClientResponse
}

func (n *Network) assertStarted(nodeIndex int) {
	if !n.Nodes[nodeIndex].Started() {
		panic(fmt.Errorf("accessing a stopped node %d", nodeIndex))
	}
}

func (n *Network) Destroy() {
	for _, node := range n.Nodes {
		node.Destroy()
	}
}

func (n *Network) MetricRegistry(nodeIndex int) metric.Registry {
	return n.Nodes[nodeIndex].metricRegistry
}

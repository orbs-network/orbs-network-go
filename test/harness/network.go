package harness

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization/supervized"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/harness/contracts"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	gossipAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
)

type InProcessNetwork interface {
	PublicApi(nodeIndex int) services.PublicApi
	GetCounterContract() *contracts.CounterContract
}

type InProcessTestNetwork interface {
	InProcessNetwork
	GossipTransport() gossipAdapter.TamperingTransport
	Description() string
	DeployBenchmarkToken(ctx context.Context, ownerAddressIndex int)
	BlockPersistence(nodeIndex int) blockStorageAdapter.InMemoryBlockPersistence
	SendTransfer(ctx context.Context, nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) chan *client.SendTransactionResponse
	SendTransferInBackground(ctx context.Context, nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) primitives.Sha256
	SendInvalidTransfer(ctx context.Context, nodeIndex int, fromAddressIndex int, toAddressIndex int) chan *client.SendTransactionResponse
	CallGetBalance(ctx context.Context, nodeIndex int, forAddressIndex int) chan uint64
	DumpState()
	WaitForTransactionInState(ctx context.Context, nodeIndex int, txhash primitives.Sha256)
	WaitForTransactionInStateForAtMost(ctx context.Context, nodeIndex int, txhash primitives.Sha256, atMost time.Duration)
	Size() int
	MetricsString(nodeIndex int) string
}

type inProcessNetwork struct {
	nodes           []*networkNode
	gossipTransport gossipAdapter.TamperingTransport
	description     string
	testLogger      log.BasicLogger
}

func (n *inProcessNetwork) StartNodes(ctx context.Context) InProcessNetwork {
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

func (n *inProcessNetwork) WaitForTransactionInState(ctx context.Context, nodeIndex int, txhash primitives.Sha256) {
	n.WaitForTransactionInStateForAtMost(ctx, nodeIndex, txhash, 1*time.Second)
}

func (n *inProcessNetwork) WaitForTransactionInStateForAtMost(ctx context.Context, nodeIndex int, txhash primitives.Sha256, atMost time.Duration) {
	blockHeight := n.BlockPersistence(nodeIndex).WaitForTransaction(ctx, txhash, atMost)
	err := n.nodes[nodeIndex].statePersistence.WaitUntilCommittedBlockOfHeight(ctx, blockHeight)
	if err != nil {
		test.DebugPrintGoroutineStacks() // since test timed out, help find deadlocked goroutines
		panic(fmt.Sprintf("statePersistence.WaitUntilCommittedBlockOfHeight failed: %s", err.Error()))
	}
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

func (n *inProcessNetwork) DeployBenchmarkToken(ctx context.Context, ownerAddressIndex int) {
	tx := <-n.SendTransfer(ctx, 0, 0, ownerAddressIndex, ownerAddressIndex) // deploy BenchmarkToken by running an empty transaction
	for i := range n.nodes {
		n.WaitForTransactionInState(ctx, i, tx.TransactionReceipt().Txhash())
	}
}

func (n *inProcessNetwork) SendTransfer(ctx context.Context, nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) chan *client.SendTransactionResponse {
	signerKeyPair := keys.Ed25519KeyPairForTests(fromAddressIndex)
	targetAddress := builders.AddressForEd25519SignerForTests(toAddressIndex)
	request := (&client.SendTransactionRequestBuilder{
		SignedTransaction: builders.TransferTransaction().WithEd25519Signer(signerKeyPair).WithAmountAndTargetAddress(amount, targetAddress).Builder(),
	}).Build()

	ch := make(chan *client.SendTransactionResponse)
	supervized.ShortLived(n.testLogger, func() {
		publicApi := n.nodes[nodeIndex].nodeLogic.PublicApi()
		output, err := publicApi.SendTransaction(ctx, &services.SendTransactionInput{
			ClientRequest: request,
		})
		if err != nil {
			panic(fmt.Sprintf("error in transfer: %v", err)) // TODO: improve
		}
		ch <- output.ClientResponse

	})
	return ch
}

// TODO: when publicApi supports returning as soon as SendTransaction is in the pool, switch to blocking implementation that waits for this
func (n *inProcessNetwork) SendTransferInBackground(ctx context.Context, nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) primitives.Sha256 {
	signerKeyPair := keys.Ed25519KeyPairForTests(fromAddressIndex)
	targetAddress := builders.AddressForEd25519SignerForTests(toAddressIndex)
	request := (&client.SendTransactionRequestBuilder{
		SignedTransaction: builders.TransferTransaction().WithEd25519Signer(signerKeyPair).WithAmountAndTargetAddress(amount, targetAddress).Builder(),
	}).Build()

	supervized.ShortLived(n.testLogger, func() {
		publicApi := n.nodes[nodeIndex].nodeLogic.PublicApi()
		publicApi.SendTransaction(ctx, &services.SendTransactionInput{ // we ignore timeout here.
			ClientRequest: request,
		})
	})
	return digest.CalcTxHash(request.SignedTransaction().Transaction())
}

func (n *inProcessNetwork) SendInvalidTransfer(ctx context.Context, nodeIndex int, fromAddressIndex int, toAddressIndex int) chan *client.SendTransactionResponse {
	signerKeyPair := keys.Ed25519KeyPairForTests(fromAddressIndex)
	targetAddress := builders.AddressForEd25519SignerForTests(toAddressIndex)
	request := (&client.SendTransactionRequestBuilder{
		SignedTransaction: builders.TransferTransaction().WithEd25519Signer(signerKeyPair).WithInvalidAmount(targetAddress).Builder(),
	}).Build()

	ch := make(chan *client.SendTransactionResponse)
	supervized.ShortLived(n.testLogger, func() {
		publicApi := n.nodes[nodeIndex].nodeLogic.PublicApi()
		output, err := publicApi.SendTransaction(ctx, &services.SendTransactionInput{
			ClientRequest: request,
		})
		if err != nil {
			panic(fmt.Sprintf("error in invalid transfer: %v", err)) // TODO: improve
		}
		ch <- output.ClientResponse
	})
	return ch
}

func (n *inProcessNetwork) CallGetBalance(ctx context.Context, nodeIndex int, forAddressIndex int) chan uint64 {
	signerKeyPair := keys.Ed25519KeyPairForTests(forAddressIndex)
	targetAddress := builders.AddressForEd25519SignerForTests(forAddressIndex)
	request := (&client.CallMethodRequestBuilder{
		Transaction: builders.GetBalanceTransaction().WithEd25519Signer(signerKeyPair).WithTargetAddress(targetAddress).Builder().Transaction,
	}).Build()

	ch := make(chan uint64)
	supervized.ShortLived(n.testLogger, func() {
		publicApi := n.nodes[nodeIndex].nodeLogic.PublicApi()
		output, err := publicApi.CallMethod(ctx, &services.CallMethodInput{
			ClientRequest: request,
		})
		if err != nil {
			panic(fmt.Sprintf("error in get balance: %v", err)) // TODO: improve
		}
		outputArgsIterator := builders.ClientCallMethodResponseOutputArgumentsDecode(output.ClientResponse)
		ch <- outputArgsIterator.NextArguments().Uint64Value()
	})
	return ch
}

func (n *inProcessNetwork) GetCounterContract() *contracts.CounterContract {
	var apis []contracts.APIProvider
	for _, node := range n.nodes {
		apis = append(apis, node)
	}
	return &contracts.CounterContract{APIs:apis, Logger:n.testLogger}
}

func (n *inProcessNetwork) DumpState() {
	for i := range n.nodes {
		n.testLogger.Info("state dump", log.Int("node", i), log.String("data", n.nodes[i].statePersistence.Dump()))
	}
}

func (n *inProcessNetwork) Size() int {
	return len(n.nodes)
}

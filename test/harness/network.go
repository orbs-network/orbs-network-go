package harness

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	gossipAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	testNativeProcessorAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/processor/native/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type InProcessNetwork interface {
	Description() string
	DeployBenchmarkToken(ownerAddressIndex int)
	GossipTransport() gossipAdapter.TamperingTransport
	PublicApi(nodeIndex int) services.PublicApi
	BlockPersistence(nodeIndex int) blockStorageAdapter.InMemoryBlockPersistence
	SendTransfer(nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) chan *client.SendTransactionResponse
	SendTransferInBackground(nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) primitives.Sha256
	SendInvalidTransfer(nodeIndex int, fromAddressIndex int, toAddressIndex int) chan *client.SendTransactionResponse
	CallGetBalance(nodeIndex int, forAddressIndex int) chan uint64
	SendDeployCounterContract(nodeIndex int) chan *client.SendTransactionResponse
	SendCounterAdd(nodeIndex int, amount uint64) chan *client.SendTransactionResponse
	CallCounterGet(nodeIndex int) chan uint64
	DumpState()
	WaitForTransactionInState(nodeIndex int, txhash primitives.Sha256)
	Size() int
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
			n.testLogger.For(log.Node(node.name)),
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
}

func (n *inProcessNetwork) WaitForTransactionInState(nodeIndex int, txhash primitives.Sha256) {
	blockHeight := n.BlockPersistence(nodeIndex).WaitForTransaction(txhash)
	err := n.nodes[nodeIndex].statePersistence.WaitUntilCommittedBlockOfHeight(blockHeight)
	if err != nil {
		panic(fmt.Sprintf("statePersistence.WaitUntilCommittedBlockOfHeight failed: %s", err.Error()))
	}
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

func (n *inProcessNetwork) DeployBenchmarkToken(ownerAddressIndex int) {
	tx := <-n.SendTransfer(0, 0, ownerAddressIndex, ownerAddressIndex) // deploy BenchmarkToken by running an empty transaction
	for i := range n.nodes {
		n.WaitForTransactionInState(i, tx.TransactionReceipt().Txhash())
	}
}

func (n *inProcessNetwork) SendTransfer(nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) chan *client.SendTransactionResponse {
	signerKeyPair := keys.Ed25519KeyPairForTests(fromAddressIndex)
	targetAddress := builders.AddressForEd25519SignerForTests(toAddressIndex)
	request := (&client.SendTransactionRequestBuilder{
		SignedTransaction: builders.TransferTransaction().WithEd25519Signer(signerKeyPair).WithAmountAndTargetAddress(amount, targetAddress).Builder(),
	}).Build()

	ch := make(chan *client.SendTransactionResponse)
	go func() {
		publicApi := n.nodes[nodeIndex].nodeLogic.PublicApi()
		output, err := publicApi.SendTransaction(&services.SendTransactionInput{
			ClientRequest: request,
		})
		if err != nil {
			panic(fmt.Sprintf("error in transfer: %v", err)) // TODO: improve
		}
		ch <- output.ClientResponse
	}()
	return ch
}

// TODO: when publicApi supports returning as soon as SendTransaction is in the pool, switch to blocking implementation that waits for this
func (n *inProcessNetwork) SendTransferInBackground(nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) primitives.Sha256 {
	signerKeyPair := keys.Ed25519KeyPairForTests(fromAddressIndex)
	targetAddress := builders.AddressForEd25519SignerForTests(toAddressIndex)
	request := (&client.SendTransactionRequestBuilder{
		SignedTransaction: builders.TransferTransaction().WithEd25519Signer(signerKeyPair).WithAmountAndTargetAddress(amount, targetAddress).Builder(),
	}).Build()

	go func() {
		publicApi := n.nodes[nodeIndex].nodeLogic.PublicApi()
		publicApi.SendTransaction(&services.SendTransactionInput{ // we ignore timeout here.
			ClientRequest: request,
		})
	}()
	return digest.CalcTxHash(request.SignedTransaction().Transaction())
}

func (n *inProcessNetwork) SendInvalidTransfer(nodeIndex int, fromAddressIndex int, toAddressIndex int) chan *client.SendTransactionResponse {
	signerKeyPair := keys.Ed25519KeyPairForTests(fromAddressIndex)
	targetAddress := builders.AddressForEd25519SignerForTests(toAddressIndex)
	request := (&client.SendTransactionRequestBuilder{
		SignedTransaction: builders.TransferTransaction().WithEd25519Signer(signerKeyPair).WithInvalidAmount(targetAddress).Builder(),
	}).Build()

	ch := make(chan *client.SendTransactionResponse)
	go func() {
		publicApi := n.nodes[nodeIndex].nodeLogic.PublicApi()
		output, err := publicApi.SendTransaction(&services.SendTransactionInput{
			ClientRequest: request,
		})
		if err != nil {
			panic(fmt.Sprintf("error in invalid transfer: %v", err)) // TODO: improve
		}
		ch <- output.ClientResponse
	}()
	return ch
}

func (n *inProcessNetwork) CallGetBalance(nodeIndex int, forAddressIndex int) chan uint64 {
	signerKeyPair := keys.Ed25519KeyPairForTests(forAddressIndex)
	targetAddress := builders.AddressForEd25519SignerForTests(forAddressIndex)
	request := (&client.CallMethodRequestBuilder{
		Transaction: builders.GetBalanceTransaction().WithEd25519Signer(signerKeyPair).WithTargetAddress(targetAddress).Builder().Transaction,
	}).Build()

	ch := make(chan uint64)
	go func() {
		publicApi := n.nodes[nodeIndex].nodeLogic.PublicApi()
		output, err := publicApi.CallMethod(&services.CallMethodInput{
			ClientRequest: request,
		})
		if err != nil {
			panic(fmt.Sprintf("error in get balance: %v", err)) // TODO: improve
		}
		outputArgsIterator := builders.ClientCallMethodResponseOutputArgumentsDecode(output.ClientResponse)
		ch <- outputArgsIterator.NextArguments().Uint64Value()
	}()
	return ch
}

func (n *inProcessNetwork) SendDeployCounterContract(nodeIndex int) chan *client.SendTransactionResponse {
	counterStart := contracts.MOCK_COUNTER_CONTRACT_START_FROM

	// if needed, provide a fake implementation of this contract to all nodes
	for _, node := range n.nodes {
		if fakeCompiler, ok := node.nativeCompiler.(testNativeProcessorAdapter.FakeCompiler); ok {
			fakeCompiler.ProvideFakeContract(contracts.MockForCounter(), string(contracts.NativeSourceCodeForCounter(counterStart)))
		}
	}

	tx := builders.Transaction().
		WithMethod("_Deployments", "deployService").
		WithArgs(
			fmt.Sprintf("CounterFrom%d", counterStart),
			uint32(protocol.PROCESSOR_TYPE_NATIVE),
			[]byte(contracts.NativeSourceCodeForCounter(counterStart)),
		)
	request := (&client.SendTransactionRequestBuilder{
		SignedTransaction: tx.Builder(),
	}).Build()

	ch := make(chan *client.SendTransactionResponse)
	go func() {
		publicApi := n.nodes[nodeIndex].nodeLogic.PublicApi()
		output, err := publicApi.SendTransaction(&services.SendTransactionInput{
			ClientRequest: request,
		})
		if err != nil {
			panic(fmt.Sprintf("error sending counter deploy: %v", err)) // TODO: improve
		}
		ch <- output.ClientResponse
	}()
	return ch
}

func (n *inProcessNetwork) SendCounterAdd(nodeIndex int, amount uint64) chan *client.SendTransactionResponse {
	counterStart := contracts.MOCK_COUNTER_CONTRACT_START_FROM

	tx := builders.Transaction().
		WithMethod(primitives.ContractName(fmt.Sprintf("CounterFrom%d", counterStart)), "add").
		WithArgs(amount)
	request := (&client.SendTransactionRequestBuilder{
		SignedTransaction: tx.Builder(),
	}).Build()

	ch := make(chan *client.SendTransactionResponse)
	go func() {
		publicApi := n.nodes[nodeIndex].nodeLogic.PublicApi()
		output, err := publicApi.SendTransaction(&services.SendTransactionInput{
			ClientRequest: request,
		})
		if err != nil {
			panic(fmt.Sprintf("error sending counter add for the amount %d: %v", amount, err)) // TODO: improve
		}
		ch <- output.ClientResponse
	}()
	return ch
}

func (n *inProcessNetwork) CallCounterGet(nodeIndex int) chan uint64 {
	counterStart := contracts.MOCK_COUNTER_CONTRACT_START_FROM

	request := (&client.CallMethodRequestBuilder{
		Transaction: &protocol.TransactionBuilder{
			ContractName: primitives.ContractName(fmt.Sprintf("CounterFrom%d", counterStart)),
			MethodName:   "get",
		},
	}).Build()

	ch := make(chan uint64)
	go func() {
		publicApi := n.nodes[nodeIndex].nodeLogic.PublicApi()
		output, err := publicApi.CallMethod(&services.CallMethodInput{
			ClientRequest: request,
		})
		if err != nil {
			panic(fmt.Sprintf("error in calling counter get: %v", err)) // TODO: improve
		}
		outputArgsIterator := builders.ClientCallMethodResponseOutputArgumentsDecode(output.ClientResponse)
		ch <- outputArgsIterator.NextArguments().Uint64Value()
	}()
	return ch
}

func (n *inProcessNetwork) DumpState() {
	for i := range n.nodes {
		n.testLogger.Info("state dump", log.Int("node", i), log.String("data", n.nodes[i].statePersistence.Dump()))
	}
}

func (n *inProcessNetwork) Size() int {
	return len(n.nodes)
}

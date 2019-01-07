package harness

import (
	"context"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/bootstrap/inmemory"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	ethereumAdapter "github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-network-go/test/harness/callcontract"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	testGossipAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/processor/native/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type TestNetworkDriver interface {
	inmemory.NetworkDriver
	BenchmarkTokenContract() callcontract.BenchmarkTokenClient
	TransportTamperer() testGossipAdapter.Tamperer
	EthereumSimulator() *ethereumAdapter.EthereumSimulator
	Description() string
	BlockPersistence(nodeIndex int) blockStorageAdapter.InMemoryBlockPersistence
	DumpState()
	WaitForTransactionInNodeState(ctx context.Context, txHash primitives.Sha256, nodeIndex int)
	MockContract(fakeContractInfo *sdkContext.ContractInfo, code string)
}

type acceptanceNetwork struct {
	inmemory.Network

	tamperingTransport testGossipAdapter.Tamperer
	description        string
	ethereumConnection *ethereumAdapter.EthereumSimulator
}

func (n *acceptanceNetwork) Start(ctx context.Context, numOfNodesToStart int) {
	n.CreateAndStartNodes(ctx, numOfNodesToStart) // needs to start first so that nodes can register their listeners to it
	n.WaitUntilReadyForTransactions(ctx)          // this is so that no transactions are sent before each node has committed block 0, otherwise transactions will be rejected
}

func (n *acceptanceNetwork) WaitForTransactionInNodeState(ctx context.Context, txHash primitives.Sha256, nodeIndex int) {
	n.Nodes[nodeIndex].WaitForTransactionInState(ctx, txHash)
}

func (n *acceptanceNetwork) Description() string {
	return n.description
}

func (n *acceptanceNetwork) TransportTamperer() testGossipAdapter.Tamperer {
	return n.tamperingTransport
}

func (n *acceptanceNetwork) EthereumSimulator() *ethereumAdapter.EthereumSimulator {
	return n.ethereumConnection
}

func (n *acceptanceNetwork) BlockPersistence(nodeIndex int) blockStorageAdapter.InMemoryBlockPersistence {
	return n.GetBlockPersistence(nodeIndex)
}

func (n *acceptanceNetwork) BenchmarkTokenContract() callcontract.BenchmarkTokenClient {
	return callcontract.NewContractClient(n)
}

func (n *acceptanceNetwork) DumpState() {
	for i := range n.Nodes {
		n.Logger.Info("state dump", log.Int("node", i), log.String("data", n.GetStatePersistence(i).Dump()))
	}
}

func (n *acceptanceNetwork) MockContract(fakeContractInfo *sdkContext.ContractInfo, code string) {
	// if needed, provide a fake implementation of this contract to all nodes
	for _, node := range n.Nodes {
		if fakeCompiler, ok := node.GetCompiler().(nativeProcessorAdapter.FakeCompiler); ok {
			fakeCompiler.ProvideFakeContract(fakeContractInfo, code)
		}
	}
}
func (n *acceptanceNetwork) Destroy() {
	for _, node := range n.Nodes {
		node.Destroy()
	}
}

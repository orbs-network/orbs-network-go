package acceptance

import (
	"context"
	"fmt"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/bootstrap/inmemory"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/services/blockstorage/adapter/testkit"
	ethereumAdapter "github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	testGossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter/testkit"
	"github.com/orbs-network/orbs-network-go/services/processor/native/adapter/fake"
	testStateStorageAdapter "github.com/orbs-network/orbs-network-go/services/statestorage/adapter/testkit"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test/acceptance/callcontract"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"time"
)

type NetworkHarness struct {
	inmemory.Network

	tamperingTransport                 testGossipAdapter.Tamperer
	ethereumConnection                 *ethereumAdapter.EthereumSimulator
	fakeCompiler                       fake.FakeCompiler
	tamperingBlockPersistences         []blockStorageAdapter.TamperingInMemoryBlockPersistence
	dumpingStatePersistences           []testStateStorageAdapter.DumpingStatePersistence
	stateBlockHeightTrackers           []*synchronization.BlockTracker
	transactionPoolBlockHeightTrackers []*synchronization.BlockTracker
}

func (n *NetworkHarness) WaitForTransactionInNodeState(ctx context.Context, txHash primitives.Sha256, nodeIndex int) {
	blockHeight := n.tamperingBlockPersistences[nodeIndex].WaitForTransaction(ctx, txHash)
	err := n.stateBlockHeightTrackers[nodeIndex].WaitForBlock(ctx, blockHeight)
	if err != nil {
		instrumentation.DebugPrintGoroutineStacks(n.Logger) // since test timed out, help find deadlocked goroutines
		panic(fmt.Sprintf("statePersistence.WaitUntilCommittedBlockOfHeight failed: %s", err.Error()))
	}
}

func (n *NetworkHarness) WaitForTransactionInState(ctx context.Context, txHash primitives.Sha256) {
	for i, node := range n.Nodes {
		if node.Started() {
			n.WaitForTransactionInNodeState(ctx, txHash, i)
		}
	}
}

func (n *NetworkHarness) TransportTamperer() testGossipAdapter.Tamperer {
	return n.tamperingTransport
}

func (n *NetworkHarness) EthereumSimulator() *ethereumAdapter.EthereumSimulator {
	return n.ethereumConnection
}

func (n *NetworkHarness) DeployBenchmarkTokenContract(ctx context.Context, ownerAddressIndex int) callcontract.BenchmarkTokenClient {
	bt := callcontract.NewContractClient(n)

	benchmarkDeploymentTimeout := 1 * time.Second
	timeoutCtx, cancel := context.WithTimeout(ctx, benchmarkDeploymentTimeout)
	defer cancel()

	res, txHash := bt.Transfer(timeoutCtx, 0, 0, ownerAddressIndex, ownerAddressIndex) // deploy BenchmarkToken by running an empty transaction

	switch res.TransactionStatus() {
	case protocol.TRANSACTION_STATUS_COMMITTED, protocol.TRANSACTION_STATUS_PENDING, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING:
		n.WaitForTransactionInState(ctx, txHash)
	default:
		panic(fmt.Sprintf("error sending transaction response: %s", res.String()))
	}
	return bt
}

func (n *NetworkHarness) MockContract(fakeContractInfo *sdkContext.ContractInfo, code string) {
	n.fakeCompiler.ProvideFakeContract(fakeContractInfo, code)
}

func (n *NetworkHarness) GetTransactionPoolBlockHeightTracker(nodeIndex int) *synchronization.BlockTracker {
	return n.transactionPoolBlockHeightTrackers[nodeIndex]
}

func (n *NetworkHarness) BlockPersistence(nodeIndex int) blockStorageAdapter.TamperingInMemoryBlockPersistence {
	return n.tamperingBlockPersistences[nodeIndex]
}

func (n *NetworkHarness) GetStatePersistence(i int) testStateStorageAdapter.DumpingStatePersistence {
	return n.dumpingStatePersistences[i]
}

func (n *NetworkHarness) Size() int {
	return len(n.Nodes)
}

func (n *NetworkHarness) DumpState() {
	for i := range n.Nodes {
		n.Logger.Info("state dump", log.Int("node", i), log.String("data", n.GetStatePersistence(i).Dump()))
	}
}

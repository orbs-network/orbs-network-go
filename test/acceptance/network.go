// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package acceptance

import (
	"context"
	"fmt"
	"github.com/orbs-network/govnr"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/bootstrap/inmemory"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/services/blockstorage/adapter/testkit"
	ethereumAdapter "github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	memoryGossip "github.com/orbs-network/orbs-network-go/services/gossip/adapter/memory"
	gossipTestAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter/testkit"
	testGossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter/testkit"
	"github.com/orbs-network/orbs-network-go/services/management"
	managementAdapter "github.com/orbs-network/orbs-network-go/services/management/adapter"
	"github.com/orbs-network/orbs-network-go/services/processor/native/adapter/fake"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/services/processor/native/adapter/fake"
	harnessStateStorageAdapter "github.com/orbs-network/orbs-network-go/services/statestorage/adapter/testkit"
	testStateStorageAdapter "github.com/orbs-network/orbs-network-go/services/statestorage/adapter/testkit"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test/acceptance/callcontract"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/scribe/log"
	"math"
	"testing"
	"time"
)

type Network struct {
	inmemory.Network

	tamperingTransport                 testGossipAdapter.Tamperer
	committeeProvider                  *managementAdapter.MemoryProvider
	ethereumConnection                 *ethereumAdapter.NopEthereumAdapter
	fakeCompiler                       *fake.FakeCompiler
	tamperingBlockPersistences         []blockStorageAdapter.TamperingInMemoryBlockPersistence
	dumpingStatePersistences           []testStateStorageAdapter.DumpingStatePersistence
	stateBlockHeightTrackers           []*synchronization.BlockTracker
	transactionPoolBlockHeightTrackers []*synchronization.BlockTracker
}

func usingABenchmarkConsensusNetwork(tb testing.TB, f func(ctx context.Context, network *Network)) {
	logger := log.DefaultTestingLogger(tb)
	ctx, cancel := context.WithCancel(context.Background())
	govnr.Recover(logfields.GovnrErrorer(logger), func() {
		defer cancel()
		network := newAcceptanceTestNetwork(ctx, logger, consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS, nil, 2, DEFAULT_ACCEPTANCE_MAX_TX_PER_BLOCK, DEFAULT_ACCEPTANCE_REQUIRED_QUORUM_PERCENTAGE, DEFAULT_ACCEPTANCE_VIRTUAL_CHAIN_ID, DEFAULT_ACCEPTANCE_EMPTY_BLOCK_TIME, nil)
		network.CreateAndStartNodes(ctx, 2)
		f(ctx, network)
	})
}

func newAcceptanceTestNetwork(ctx context.Context, testLogger log.Logger, consensusAlgo consensus.ConsensusAlgoType, preloadedBlocks []*protocol.BlockPairContainer,
	numNodes int, maxTxPerBlock uint32, requiredQuorumPercentage uint32, vcid primitives.VirtualChainId, emptyBlockTime time.Duration,
	configOverride func(cfg config.OverridableConfig) config.OverridableConfig) *Network {

	testLogger.Info("===========================================================================")
	testLogger.Info("creating acceptance test network", log.String("consensus", consensusAlgo.String()), log.Int("num-nodes", numNodes))

	leaderKeyPair := testKeys.EcdsaSecp256K1KeyPairForTests(0)

	genesisValidatorNodes := map[string]config.ValidatorNode{}
	privateKeys := map[string]primitives.EcdsaSecp256K1PrivateKey{}
	var nodeOrder []primitives.NodeAddress
	for i := 0; i < int(numNodes); i++ {
		nodeAddress := testKeys.EcdsaSecp256K1KeyPairForTests(i).NodeAddress()
		genesisValidatorNodes[nodeAddress.KeyForMap()] = config.NewHardCodedValidatorNode(nodeAddress)
		privateKeys[nodeAddress.KeyForMap()] = testKeys.EcdsaSecp256K1KeyPairForTests(i).PrivateKey()
		nodeOrder = append(nodeOrder, nodeAddress)
	}

	var cfgTemplate config.OverridableConfig
	cfgTemplate = config.ForAcceptanceTestNetwork(
		genesisValidatorNodes,
		leaderKeyPair.NodeAddress(),
		consensusAlgo,
		maxTxPerBlock,
		requiredQuorumPercentage,
		vcid,
		emptyBlockTime,
	)

	if configOverride != nil {
		cfgTemplate = configOverride(cfgTemplate)
	}

	sharedTamperingTransport := gossipTestAdapter.NewTamperingTransport(testLogger, memoryGossip.NewTransport(ctx, testLogger, genesisValidatorNodes))
	sharedManagementProvider := managementAdapter.NewMemoryProvider(cfgTemplate, testLogger)
	sharedManagement := management.NewManagement(ctx, cfgTemplate, sharedManagementProvider, sharedTamperingTransport, testLogger)
	sharedCompiler := nativeProcessorAdapter.NewCompiler()
	sharedEthereumSimulator := &ethereumAdapter.NopEthereumAdapter{}

	var tamperingBlockPersistences []blockStorageAdapter.TamperingInMemoryBlockPersistence
	var dumpingStatePersistences []harnessStateStorageAdapter.DumpingStatePersistence
	var transactionPoolTrackers []*synchronization.BlockTracker
	var stateTrackers []*synchronization.BlockTracker

	provider := func(idx int, nodeConfig config.NodeConfig, logger log.Logger, metricRegistry metric.Registry) *inmemory.NodeDependencies {
		tamperingBlockPersistence := blockStorageAdapter.NewBlockPersistence(logger, preloadedBlocks, metricRegistry)
		dumpingStateStorage := harnessStateStorageAdapter.NewDumpingStatePersistence(metricRegistry)

		txPoolHeightTracker := synchronization.NewBlockTracker(logger, 0, math.MaxUint16)
		stateHeightTracker := synchronization.NewBlockTracker(logger, 0, math.MaxUint16)

		tamperingBlockPersistences = append(tamperingBlockPersistences, tamperingBlockPersistence)
		dumpingStatePersistences = append(dumpingStatePersistences, dumpingStateStorage)
		transactionPoolTrackers = append(transactionPoolTrackers, txPoolHeightTracker)
		stateTrackers = append(stateTrackers, stateHeightTracker)

		return &inmemory.NodeDependencies{
			BlockPersistence:                   tamperingBlockPersistence,
			StatePersistence:                   dumpingStateStorage,
			EtherConnection:                    sharedEthereumSimulator,
			Compiler:                           sharedCompiler,
			TransactionPoolBlockHeightReporter: txPoolHeightTracker,
			StateBlockHeightReporter:           stateHeightTracker,
		}
	}

	harness := &Network{
		Network:                            *inmemory.NewNetworkWithNumOfNodes(genesisValidatorNodes, nodeOrder, privateKeys, testLogger, cfgTemplate, sharedTamperingTransport, sharedManagement, nil, provider),
		tamperingTransport:                 sharedTamperingTransport,
		committeeProvider:                  sharedManagementProvider,
		ethereumConnection:                 sharedEthereumSimulator,
		fakeCompiler:                       sharedCompiler,
		tamperingBlockPersistences:         tamperingBlockPersistences,
		dumpingStatePersistences:           dumpingStatePersistences,
		stateBlockHeightTrackers:           stateTrackers,
		transactionPoolBlockHeightTrackers: transactionPoolTrackers,
	}

	return harness // call harness.CreateAndStartNodes() to launch nodes in the network
}

func (n *Network) WaitForTransactionInNodeState(ctx context.Context, txHash primitives.Sha256, nodeIndex int) {
	blockHeight := n.tamperingBlockPersistences[nodeIndex].WaitForTransaction(ctx, txHash)
	err := n.stateBlockHeightTrackers[nodeIndex].WaitForBlock(ctx, blockHeight)
	if err != nil {
		instrumentation.DebugPrintGoroutineStacks(n.Logger) // since test timed out, help find deadlocked goroutines
		panic(fmt.Sprintf("statePersistence.WaitUntilCommittedBlockOfHeight failed: %s", err.Error()))
	}
}

func (n *Network) WaitForTransactionReceiptInTransactionPool(ctx context.Context, txHash primitives.Sha256, nodeIndex int) {
	blockHeight := n.tamperingBlockPersistences[nodeIndex].WaitForTransaction(ctx, txHash)
	err := n.transactionPoolBlockHeightTrackers[nodeIndex].WaitForBlock(ctx, blockHeight)
	if err != nil {
		instrumentation.DebugPrintGoroutineStacks(n.Logger) // since test timed out, help find deadlocked goroutines
		panic(fmt.Sprintf("statePersistence.WaitForTransactionInTransactionPool failed: %s", err.Error()))
	}
}

func (n *Network) WaitForTransactionInState(ctx context.Context, txHash primitives.Sha256) {
	for i, node := range n.Nodes {
		if node.Started() {
			n.WaitForTransactionInNodeState(ctx, txHash, i)
		}
	}
}

func (n *Network) TransportTamperer() testGossipAdapter.Tamperer {
	return n.tamperingTransport
}

func (n *Network) DeployBenchmarkTokenContract(ctx context.Context, ownerAddressIndex int) callcontract.BenchmarkTokenClient {
	bt := callcontract.NewContractClient(n)

	benchmarkDeploymentTimeout := 5 * time.Second
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

func (n *Network) MockContract(fakeContractInfo *sdkContext.ContractInfo, code string) {
	n.fakeCompiler.ProvideFakeContract(fakeContractInfo, code)
}

func (n *Network) GetTransactionPoolBlockHeightTracker(nodeIndex int) *synchronization.BlockTracker {
	return n.transactionPoolBlockHeightTrackers[nodeIndex]
}

func (n *Network) BlockPersistence(nodeIndex int) blockStorageAdapter.TamperingInMemoryBlockPersistence {
	return n.tamperingBlockPersistences[nodeIndex]
}

func (n *Network) GetStatePersistence(i int) testStateStorageAdapter.DumpingStatePersistence {
	return n.dumpingStatePersistences[i]
}

func (n *Network) Size() int {
	return len(n.Nodes)
}

func (n *Network) DumpState() {
	for i := range n.Nodes {
		n.Logger.Info("state dump", log.Int("node", i), log.String("data", n.GetStatePersistence(i).Dump()))
	}
}

func (n *Network) WaitForBlock(ctx context.Context, height primitives.BlockHeight) {
	for _, tracker := range n.transactionPoolBlockHeightTrackers {
		tracker.WaitForBlock(ctx, height)
	}
}

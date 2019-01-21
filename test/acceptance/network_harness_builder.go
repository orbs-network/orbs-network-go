package acceptance

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap/inmemory"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/services/blockstorage/adapter/testkit"
	ethereumAdapter "github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	memoryGossip "github.com/orbs-network/orbs-network-go/services/gossip/adapter/memory"
	gossipTestAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter/testkit"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/services/processor/native/adapter/fake"
	harnessStateStorageAdapter "github.com/orbs-network/orbs-network-go/services/statestorage/adapter/testkit"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-network-go/test"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/pkg/errors"
	"math"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

type testContext interface {
	canFail
	subTester
}

type benchContext interface {
	canFail
	subBencher
}

type canFail interface {
	Failed() bool
	Fatal(args ...interface{})
}

type subTester interface {
	Name() string
	Run(name string, f func(t *testing.T)) bool
}

type subBencher interface {
	Name() string
	Run(name string, f func(b *testing.B)) bool
}

var ENABLE_LEAN_HELIX_IN_ACCEPTANCE_TESTS = false

type networkHarnessBuilder struct {
	f                        canFail
	st                       subTester
	sb                       subBencher
	numNodes                 int
	consensusAlgos           []consensus.ConsensusAlgoType
	testId                   string
	setupFunc                func(ctx context.Context, network NetworkHarness)
	logFilters               []log.Filter
	maxTxPerBlock            uint32
	allowedErrors            []string
	numOfNodesToStart        int
	requiredQuorumPercentage uint32
}

// TODO Make the "primary consensus algo" configurable https://tree.taiga.io/project/orbs-network/us/632
func newHarness(t testContext) *networkHarnessBuilder {
	n := &networkHarnessBuilder{f: t.(canFail), st: t.(subTester), maxTxPerBlock: 30, requiredQuorumPercentage: 100}

	var algos []consensus.ConsensusAlgoType
	if ENABLE_LEAN_HELIX_IN_ACCEPTANCE_TESTS {
		algos = []consensus.ConsensusAlgoType{consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX, consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS}
	} else {
		algos = []consensus.ConsensusAlgoType{consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS}
	}

	return n.
		WithTestId(getCallerFuncName()).
		WithNumNodes(4).
		WithConsensusAlgos(algos...).
		AllowingErrors("ValidateBlockProposal failed.*") // it is acceptable for validation to fail in one or more nodes, as long as f+1 nodes are in agreement on a block and even if they do not, a new leader should eventually be able to reach consensus on the block
}

func newBenchHarness(b benchContext) *networkHarnessBuilder {
	n := &networkHarnessBuilder{f: b.(canFail), sb: b.(subBencher), maxTxPerBlock: 30, requiredQuorumPercentage: 100}

	var algos []consensus.ConsensusAlgoType
	if ENABLE_LEAN_HELIX_IN_ACCEPTANCE_TESTS {
		algos = []consensus.ConsensusAlgoType{consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX, consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS}
	} else {
		algos = []consensus.ConsensusAlgoType{consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS}
	}

	return n.
		WithTestId(getCallerFuncName()).
		WithNumNodes(4).
		WithConsensusAlgos(algos...).
		AllowingErrors("ValidateBlockProposal failed.*") // it is acceptable for validation to fail in one or more nodes, as long as f+1 nodes are in agreement on a block and even if they do not, a new leader should eventually be able to reach consensus on the block
}

func (b *networkHarnessBuilder) WithLogFilters(filters ...log.Filter) *networkHarnessBuilder {
	b.logFilters = filters
	return b
}

func (b *networkHarnessBuilder) WithTestId(testId string) *networkHarnessBuilder {
	randNum := rand.Intn(1000000)
	b.testId = "acceptance-" + testId + "-" + strconv.FormatInt(time.Now().Unix(), 10) + "-" + strconv.FormatInt(int64(randNum), 10)
	return b
}

func (b *networkHarnessBuilder) WithNumNodes(numNodes int) *networkHarnessBuilder {
	b.numNodes = numNodes
	return b
}

func (b *networkHarnessBuilder) WithConsensusAlgos(algos ...consensus.ConsensusAlgoType) *networkHarnessBuilder {
	b.consensusAlgos = algos
	return b
}

// setup runs when all adapters have been created but before the nodes are started
func (b *networkHarnessBuilder) WithSetup(f func(ctx context.Context, network NetworkHarness)) *networkHarnessBuilder {
	b.setupFunc = f
	return b
}

func (b *networkHarnessBuilder) WithMaxTxPerBlock(maxTxPerBlock uint32) *networkHarnessBuilder {
	b.maxTxPerBlock = maxTxPerBlock
	return b
}

func (b *networkHarnessBuilder) AllowingErrors(allowedErrors ...string) *networkHarnessBuilder {
	b.allowedErrors = append(b.allowedErrors, allowedErrors...)
	return b
}

func (b *networkHarnessBuilder) Start(f func(ctx context.Context, network NetworkHarness)) {
	b.StartWithRestart(func(ctx context.Context, network NetworkHarness, _ func() NetworkHarness) {
		f(ctx, network)
	})
}

func (b *networkHarnessBuilder) StartWithRestart(f func(ctx context.Context, network NetworkHarness, restartPreservingBlocks func() NetworkHarness)) {
	if b.numOfNodesToStart == 0 {
		b.numOfNodesToStart = b.numNodes
	}

	for _, consensusAlgo := range b.consensusAlgos {

		restartableTest := func(ctx context.Context) {
			test.WithContextWithTimeout(15*time.Second, func(ctx context.Context) { //TODO(v1) 10 seconds is infinity; reduce to 2 seconds when system is more stable (after we add feature of custom config per test)
				networkCtx, cancelNetwork := context.WithCancel(ctx)
				testId := b.testId + "-" + consensusAlgo.String()
				logger, errorRecorder := b.makeLogger(testId)
				network := b.newAcceptanceTestNetwork(networkCtx, logger, consensusAlgo, nil)

				logger.Info("acceptance network created")
				defer printTestIdOnFailure(b.f, testId)
				defer dumpStateOnFailure(b.f, network)
				defer test.RequireNoUnexpectedErrors(b.f, errorRecorder)

				if b.setupFunc != nil {
					b.setupFunc(networkCtx, network)
				}

				network.CreateAndStartNodes(networkCtx, b.numOfNodesToStart)
				logger.Info("acceptance network started")

				restart := func() NetworkHarness {
					cancelNetwork()
					network.Destroy()
					time.Sleep(5 * time.Millisecond) // give context dependent goroutines 5 ms to terminate gracefully

					// signal the old network to stop
					networkCtx, cancelNetwork = context.WithCancel(ctx) // allocate new cancel func for new network
					newNetwork := b.newAcceptanceTestNetwork(ctx, logger, consensusAlgo, extractBlocks(network.BlockPersistence(0)))
					logger.Info("acceptance network re-created")

					newNetwork.CreateAndStartNodes(networkCtx, b.numOfNodesToStart)
					logger.Info("acceptance network re-started")

					return newNetwork
				}

				logger.Info("acceptance network running test")
				f(ctx, network, restart)
				time.Sleep(5 * time.Millisecond) // give context dependent goroutines 5 ms to terminate gracefully
			}
		}

		if b.sb != nil {
			b.sb.Run(consensusAlgo.String(), func(t *testing.B) {
				test.WithContextWithTimeout(15*time.Second, restartableTest)
			})
		} else {
			b.st.Run(consensusAlgo.String(), func(t *testing.T) {
				test.WithContextWithTimeout(15*time.Second, restartableTest)
			})
		}
	}
}

func extractBlocks(blocks blockStorageAdapter.TamperingInMemoryBlockPersistence) []*protocol.BlockPairContainer {
	lastBlock, err := blocks.GetLastBlock()
	if err != nil {
		panic(errors.Wrapf(err, "spawn network: failed reading block height"))
	}
	var blockPairs []*protocol.BlockPairContainer
	pageSize := uint8(lastBlock.ResultsBlock.Header.BlockHeight())
	err = blocks.ScanBlocks(1, pageSize, func(first primitives.BlockHeight, page []*protocol.BlockPairContainer) bool {
		blockPairs = page // TODO should we copy the slice here to make sure both networks are isolated?
		return false
	})
	if err != nil {
		panic(errors.Wrapf(err, "spawn network: failed extract blocks"))
	}
	return blockPairs
}

func (b *networkHarnessBuilder) makeLogger(testId string) (log.BasicLogger, test.ErrorTracker) {
	errorRecorder := log.NewErrorRecordingOutput(b.allowedErrors)
	logger := log.GetLogger(
		log.String("_test", "acceptance"),
		log.String("_branch", os.Getenv("GIT_BRANCH")),
		log.String("_commit", os.Getenv("GIT_COMMIT")),
		log.String("_test-id", testId)).
		WithOutput(makeFormattingOutput(testId), errorRecorder).
		WithFilters(b.logFilters...)
	//WithFilters(log.Or(log.OnlyErrors(), log.OnlyCheckpoints(), log.OnlyMetrics()))

	return logger, errorRecorder
}

func (b *networkHarnessBuilder) WithNumRunningNodes(numNodes int) *networkHarnessBuilder {
	b.numOfNodesToStart = numNodes
	return b
}

func (b *networkHarnessBuilder) WithRequiredQuorumPercentage(percentage int) *networkHarnessBuilder {
	b.requiredQuorumPercentage = uint32(percentage)
	return b
}

func (b *networkHarnessBuilder) newAcceptanceTestNetwork(ctx context.Context, testLogger log.BasicLogger, consensusAlgo consensus.ConsensusAlgoType, preloadedBlocks []*protocol.BlockPairContainer) *networkHarness {

	testLogger.Info("===========================================================================")
	testLogger.Info("creating acceptance test network", log.String("consensus", consensusAlgo.String()), log.Int("num-nodes", b.numNodes))

	leaderKeyPair := testKeys.EcdsaSecp256K1KeyPairForTests(0)

	federationNodes := map[string]config.FederationNode{}
	privateKeys := map[string]primitives.EcdsaSecp256K1PrivateKey{}
	var nodeOrder []primitives.NodeAddress
	for i := 0; i < int(b.numNodes); i++ {
		nodeAddress := testKeys.EcdsaSecp256K1KeyPairForTests(i).NodeAddress()
		federationNodes[nodeAddress.KeyForMap()] = config.NewHardCodedFederationNode(nodeAddress)
		privateKeys[nodeAddress.KeyForMap()] = testKeys.EcdsaSecp256K1KeyPairForTests(i).PrivateKey()
		nodeOrder = append(nodeOrder, nodeAddress)
	}

	cfgTemplate := config.ForAcceptanceTestNetwork(
		federationNodes,
		leaderKeyPair.NodeAddress(),
		consensusAlgo,
		b.maxTxPerBlock,
		b.requiredQuorumPercentage,
	)

	sharedTamperingTransport := gossipTestAdapter.NewTamperingTransport(testLogger, memoryGossip.NewTransport(ctx, testLogger, federationNodes))
	sharedCompiler := nativeProcessorAdapter.NewCompiler()
	sharedEthereumSimulator := ethereumAdapter.NewEthereumSimulatorConnection(testLogger)

	var tamperingBlockPersistences []blockStorageAdapter.TamperingInMemoryBlockPersistence
	var dumpingStatePersistences []harnessStateStorageAdapter.DumpingStatePersistence
	var transactionPoolTrackers []*synchronization.BlockTracker
	var stateTrackers []*synchronization.BlockTracker

	provider := func(idx int, nodeConfig config.NodeConfig, logger log.BasicLogger, metricRegistry metric.Registry) *inmemory.NodeDependencies {
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

	harness := &networkHarness{
		Network:                            *inmemory.NewNetworkWithNumOfNodes(federationNodes, nodeOrder, privateKeys, testLogger, cfgTemplate, sharedTamperingTransport, provider),
		tamperingTransport:                 sharedTamperingTransport,
		ethereumConnection:                 sharedEthereumSimulator,
		fakeCompiler:                       sharedCompiler,
		tamperingBlockPersistences:         tamperingBlockPersistences,
		dumpingStatePersistences:           dumpingStatePersistences,
		stateBlockHeightTrackers:           stateTrackers,
		transactionPoolBlockHeightTrackers: transactionPoolTrackers,
	}

	return harness // call harness.CreateAndStartNodes() to launch nodes in the network
}

func makeFormattingOutput(testId string) log.Output {
	var output log.Output
	if os.Getenv("NO_LOG_STDOUT") == "true" {
		logFile, err := os.OpenFile(config.GetProjectSourceRootPath()+"/_logs/acceptance/"+testId+".log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}

		output = log.NewFormattingOutput(logFile, log.NewJsonFormatter())
	} else {
		output = log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter())
	}
	return output
}

func printTestIdOnFailure(f canFail, testId string) {
	if f.Failed() {
		fmt.Println("FAIL search snippet: grep _test-id="+testId, "test.out")
	}
}

func dumpStateOnFailure(f canFail, network NetworkHarness) {
	if f.Failed() {
		network.DumpState()
	}
}

func getCallerFuncName() string {
	pc, _, _, _ := runtime.Caller(2)
	packageAndFuncName := runtime.FuncForPC(pc).Name()
	parts := strings.Split(packageAndFuncName, ".")
	return parts[len(parts)-1]
}

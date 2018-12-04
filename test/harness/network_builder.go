package harness

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap/inmemory"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	ethereumAdapter "github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	gossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/test"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	gossipTestAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/processor/native/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/pkg/errors"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type canFailAndLog interface {
	test.NamedLogger
	Failed() bool
	Fatal(args ...interface{})
}

type acceptanceTestNetworkBuilder struct {
	fl                       canFailAndLog
	numNodes                 int
	consensusAlgos           []consensus.ConsensusAlgoType
	testId                   string
	setupFunc                func(ctx context.Context, network TestNetworkDriver)
	logFilters               []log.Filter
	maxTxPerBlock            uint32
	allowedErrors            []string
	numOfNodesToStart        int
	requiredQuorumPercentage uint32
}

func Network(fl canFailAndLog) *acceptanceTestNetworkBuilder {
	n := &acceptanceTestNetworkBuilder{fl: fl, maxTxPerBlock: 30, requiredQuorumPercentage: 100}

	return n.
		WithTestId(getCallerFuncName()).
		WithNumNodes(2).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS)
}

func (b *acceptanceTestNetworkBuilder) WithLogFilters(filters ...log.Filter) *acceptanceTestNetworkBuilder {
	b.logFilters = filters
	return b
}

func (b *acceptanceTestNetworkBuilder) WithTestId(testId string) *acceptanceTestNetworkBuilder {
	b.testId = "acceptance-" + testId + "-" + strconv.FormatInt(time.Now().Unix(), 10)
	return b
}

func (b *acceptanceTestNetworkBuilder) WithNumNodes(numNodes int) *acceptanceTestNetworkBuilder {
	b.numNodes = numNodes
	return b
}

func (b *acceptanceTestNetworkBuilder) WithConsensusAlgos(algos ...consensus.ConsensusAlgoType) *acceptanceTestNetworkBuilder {
	b.consensusAlgos = algos
	return b
}

// setup runs when all adapters have been created but before the nodes are started
func (b *acceptanceTestNetworkBuilder) WithSetup(f func(ctx context.Context, network TestNetworkDriver)) *acceptanceTestNetworkBuilder {
	b.setupFunc = f
	return b
}

func (b *acceptanceTestNetworkBuilder) WithMaxTxPerBlock(maxTxPerBlock uint32) *acceptanceTestNetworkBuilder {
	b.maxTxPerBlock = maxTxPerBlock
	return b
}

func (b *acceptanceTestNetworkBuilder) AllowingErrors(allowedErrors ...string) *acceptanceTestNetworkBuilder {
	b.allowedErrors = append(b.allowedErrors, allowedErrors...)
	return b
}

func (b *acceptanceTestNetworkBuilder) Start(f func(ctx context.Context, network TestNetworkDriver)) {
	b.StartWithRestart(func(ctx context.Context, network TestNetworkDriver, _ func() TestNetworkDriver) {
		f(ctx, network)
	})
}

func (b *acceptanceTestNetworkBuilder) StartWithRestart(f func(ctx context.Context, network TestNetworkDriver, restartPreservingBlocks func() TestNetworkDriver)) {
	if b.numOfNodesToStart == 0 {
		b.numOfNodesToStart = b.numNodes
	}

	for _, consensusAlgo := range b.consensusAlgos {

		// start test
		test.WithContextWithTimeout(b.fl, 10*time.Second, func(ctx context.Context) { //TODO 10 seconds is infinity; reduce to 2 seconds when system is more stable (after we add feature of custom config per test)
			networkCtx, cancelNetwork := context.WithCancel(ctx)
			testId := b.testId + "-" + consensusAlgo.String()
			logger, errorRecorder := b.makeLogger(testId)
			network := b.newAcceptanceTestNetwork(networkCtx, logger, consensusAlgo, nil)

			defer printTestIdOnFailure(b.fl, testId)
			defer dumpStateOnFailure(b.fl, network)
			defer test.RequireNoUnexpectedErrors(b.fl, errorRecorder)

			if b.setupFunc != nil {
				b.setupFunc(networkCtx, network)
			}

			network.Start(networkCtx, b.numOfNodesToStart)

			restart := func() TestNetworkDriver {
				cancelNetwork()
				network.Destroy()
				time.Sleep(5 * time.Millisecond) // give context dependent goroutines 5 ms to terminate gracefully

				// signal the old network to stop
				networkCtx, cancelNetwork = context.WithCancel(ctx) // allocate new cancel func for new network
				newNetwork := b.newAcceptanceTestNetwork(ctx, logger, consensusAlgo, extractBlocks(network.BlockPersistence(0)))
				newNetwork.Start(networkCtx, b.numOfNodesToStart)
				return newNetwork
			}

			f(ctx, network, restart)
		})
		// end test

		time.Sleep(5 * time.Millisecond) // give context dependent goroutines 5 ms to terminate gracefully
	}
}

func extractBlocks(blocks blockStorageAdapter.InMemoryBlockPersistence) []*protocol.BlockPairContainer {
	lastBlock, err := blocks.GetLastBlock()
	if err != nil {
		panic(errors.Wrapf(err, "spawn network: failed reading block height"))
	}
	blockPairs, _, _, err := blocks.GetBlocks(1, lastBlock.ResultsBlock.Header.BlockHeight()+1)
	if err != nil {
		panic(errors.Wrapf(err, "spawn network: failed extract blocks"))
	}
	return blockPairs
}

func (b *acceptanceTestNetworkBuilder) makeLogger(testId string) (log.BasicLogger, test.ErrorTracker) {
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

func (b *acceptanceTestNetworkBuilder) WithNumRunningNodes(numNodes int) *acceptanceTestNetworkBuilder {
	b.numOfNodesToStart = numNodes
	return b
}

func (b *acceptanceTestNetworkBuilder) WithRequiredQuorumPercentage(percentage int) *acceptanceTestNetworkBuilder {
	b.requiredQuorumPercentage = uint32(percentage)
	return b
}

func (b *acceptanceTestNetworkBuilder) newAcceptanceTestNetwork(ctx context.Context, testLogger log.BasicLogger, consensusAlgo consensus.ConsensusAlgoType, preloadedBlocks []*protocol.BlockPairContainer) *acceptanceNetwork {

	testLogger.Info("===========================================================================")
	testLogger.Info("creating acceptance test network", log.String("consensus", consensusAlgo.String()), log.Int("num-nodes", b.numNodes))
	description := fmt.Sprintf("network with %d nodes running %s", b.numNodes, consensusAlgo)

	leaderKeyPair := testKeys.Ed25519KeyPairForTests(0)

	federationNodes := make(map[string]config.FederationNode)
	for i := 0; i < int(b.numNodes); i++ {
		publicKey := testKeys.Ed25519KeyPairForTests(i).PublicKey()
		federationNodes[publicKey.KeyForMap()] = config.NewHardCodedFederationNode(publicKey)
	}

	cfg := config.ForAcceptanceTestNetwork(
		federationNodes,
		leaderKeyPair.PublicKey(),
		consensusAlgo,
		b.maxTxPerBlock,
		b.requiredQuorumPercentage,
	)

	sharedTamperingTransport := gossipTestAdapter.NewTamperingTransport(testLogger, gossipAdapter.NewMemoryTransport(ctx, testLogger, federationNodes))
	sharedEthereumSimulator := ethereumAdapter.NewEthereumSimulatorConnection(testLogger)

	network := &acceptanceNetwork{
		Network:            inmemory.NewNetwork(testLogger, sharedTamperingTransport, sharedEthereumSimulator),
		tamperingTransport: sharedTamperingTransport,
		ethereumConnection: sharedEthereumSimulator,
		description:        description,
	}

	for i := 0; i < b.numNodes; i++ {
		keyPair := testKeys.Ed25519KeyPairForTests(i)

		nodeCfg := cfg.OverrideNodeSpecificValues(0, keyPair.PublicKey(), keyPair.PrivateKey())

		metricRegistry := metric.NewRegistry()
		nodeLogger := testLogger.WithTags(log.Node(nodeCfg.NodePublicKey().String()))
		blockStorageAdapter := blockStorageAdapter.NewInMemoryBlockPersistenceWithBlocks(nodeLogger, preloadedBlocks, metricRegistry)

		network.AddNode(keyPair, nodeCfg, nativeProcessorAdapter.NewFakeCompiler(), blockStorageAdapter, metricRegistry, nodeLogger)
	}

	return network

	// must call network.Start(ctx) to actually start the nodes in the network
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

func printTestIdOnFailure(f canFailAndLog, testId string) {
	if f.Failed() {
		fmt.Println("FAIL search snippet: grep _test-id="+testId, "test.out")
	}
}

func dumpStateOnFailure(f canFailAndLog, network TestNetworkDriver) {
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

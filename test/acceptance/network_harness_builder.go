package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

var ENABLE_LEAN_HELIX_IN_ACCEPTANCE_TESTS = false
var TEST_TIMEOUT_HARD_LIMIT = 90 * time.Second //TODO(v1) 10 seconds is infinity; reduce to 2 seconds when system is more stable (after we add feature of custom config per test)
var DEFAULT_NODE_COUNT_FOR_ACCEPTANCE = 7

type networkHarnessBuilder struct {
	numNodes                 int
	consensusAlgos           []consensus.ConsensusAlgoType
	testId                   string
	setupFunc                func(ctx context.Context, network *NetworkHarness)
	logFilters               []log.Filter
	maxTxPerBlock            uint32
	allowedErrors            []string
	numOfNodesToStart        int
	requiredQuorumPercentage uint32
	blockChain               []*protocol.BlockPairContainer
}

func newHarness() *networkHarnessBuilder {
	n := &networkHarnessBuilder{maxTxPerBlock: 30, requiredQuorumPercentage: 100}

	var algos []consensus.ConsensusAlgoType
	if ENABLE_LEAN_HELIX_IN_ACCEPTANCE_TESTS {
		algos = []consensus.ConsensusAlgoType{consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX, consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS}
	} else {
		algos = []consensus.ConsensusAlgoType{consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS}
	}

	harness := n.
		WithTestId(getCallerFuncName()).
		WithNumNodes(DEFAULT_NODE_COUNT_FOR_ACCEPTANCE).
		WithConsensusAlgos(algos...).
		AllowingErrors(
			"ValidateBlockProposal failed.*",                         // it is acceptable for validation to fail in one or more nodes, as long as f+1 nodes are in agreement on a block and even if they do not, a new leader should eventually be able to reach consensus on the block
			"all consensus [01] algos refused to validate the block", // it is allowed for n-1 nodes to close a block, and the last node may receive this block via block sync before its consensus algos had time to register (super rare scheduling event)
		)

	return harness
}

func (b *networkHarnessBuilder) WithLogFilters(filters ...log.Filter) *networkHarnessBuilder {
	b.logFilters = filters
	return b
}

func (b *networkHarnessBuilder) WithTestId(testId string) *networkHarnessBuilder {
	randNum := rand.Intn(1000)
	b.testId = "acc-" + testId + "-" + strconv.FormatInt(time.Now().Unix(), 10) + "-" + strconv.FormatInt(int64(randNum), 10)
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
func (b *networkHarnessBuilder) WithSetup(f func(ctx context.Context, network *NetworkHarness)) *networkHarnessBuilder {
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

func (b *networkHarnessBuilder) Start(tb testing.TB, f func(tb testing.TB, ctx context.Context, network *NetworkHarness)) {
	if b.numOfNodesToStart == 0 {
		b.numOfNodesToStart = b.numNodes
	}

	for _, consensusAlgo := range b.consensusAlgos {
		switch runner := tb.(type) {
		case *testing.T:
			runner.Run(consensusAlgo.String(), func(t *testing.T) {
				b.runTest(t, consensusAlgo, f)
			})
		case *testing.B:
			runner.Run(consensusAlgo.String(), func(t *testing.B) {
				b.runTest(t, consensusAlgo, f)
			})
		default:
			panic("unexpected TB implementation")
		}
	}
}

func (b *networkHarnessBuilder) runTest(tb testing.TB, consensusAlgo consensus.ConsensusAlgoType, f func(tb testing.TB, ctx context.Context, network *NetworkHarness)) {
	testId := b.testId + "-" + toShortConsensusAlgoStr(consensusAlgo)
	logger, errorRecorder := b.makeLogger(tb, testId)

	supervised.Recover(logger, func() {

		test.WithContextWithTimeout(TEST_TIMEOUT_HARD_LIMIT, func(ctx context.Context) {
			network := newAcceptanceTestNetwork(ctx, logger, consensusAlgo, b.blockChain, b.numNodes, b.maxTxPerBlock, b.requiredQuorumPercentage)

			logger.Info("acceptance network created")
			defer printTestIdOnFailure(tb, testId)
			defer dumpStateOnFailure(tb, network)
			defer test.RequireNoUnexpectedErrors(tb, errorRecorder)

			if b.setupFunc != nil {
				b.setupFunc(ctx, network)
			}

			network.CreateAndStartNodes(ctx, b.numOfNodesToStart)
			logger.Info("acceptance network started")

			logger.Info("acceptance network running test")
			f(tb, ctx, network)
			time.Sleep(10 * time.Millisecond) // give context dependent goroutines 5 ms to terminate gracefully
		})
	})
}

func toShortConsensusAlgoStr(algoType consensus.ConsensusAlgoType) string {
	str := algoType.String()
	if len(str) < 20 {
		return str
	}
	return str[20:] // remove the "CONSENSUS_ALGO_TYPE_" prefix
}

func (b *networkHarnessBuilder) makeLogger(tb testing.TB, testId string) (log.BasicLogger, test.ErrorTracker) {

	testOutput := log.NewTestOutput(tb, log.NewHumanReadableFormatter())
	for _, pattern := range b.allowedErrors {
		testOutput.AllowErrorsMatching(pattern)
	}

	logger := log.GetLogger(
		log.String("_test", "acceptance"),
		log.String("_test-id", testId)).
		WithOutput(testOutput).
		WithFilters(log.IgnoreMessagesMatching("transport message received"), log.IgnoreMessagesMatching("Metric recorded")).
		WithFilters(b.logFilters...)
	//WithFilters(log.Or(log.OnlyErrors(), log.OnlyCheckpoints(), log.OnlyMetrics()))

	return logger, testOutput
}

func (b *networkHarnessBuilder) WithNumRunningNodes(numNodes int) *networkHarnessBuilder {
	b.numOfNodesToStart = numNodes
	return b
}

func (b *networkHarnessBuilder) WithRequiredQuorumPercentage(percentage int) *networkHarnessBuilder {
	b.requiredQuorumPercentage = uint32(percentage)
	return b
}

func (b *networkHarnessBuilder) WithInitialBlocks(blocks []*protocol.BlockPairContainer) *networkHarnessBuilder {
	b.blockChain = blocks
	return b
}

func printTestIdOnFailure(tb testing.TB, testId string) {
	if tb.Failed() {
		tb.Error("FAIL search snippet: grep _test-id="+testId, "test.out")
	}
}

func dumpStateOnFailure(tb testing.TB, network *NetworkHarness) {
	if tb.Failed() {
		network.DumpState()
	}
}

func getCallerFuncName() string {
	pc, _, _, _ := runtime.Caller(3)
	packageAndFuncName := runtime.FuncForPC(pc).Name()
	parts := strings.Split(packageAndFuncName, ".")
	return parts[len(parts)-1]
}

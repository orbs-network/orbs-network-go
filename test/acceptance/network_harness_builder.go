// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/memory"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/scribe/log"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

const ENABLE_LEAN_HELIX_IN_ACCEPTANCE_TESTS = true
const TEST_TIMEOUT_HARD_LIMIT = 20 * time.Second //TODO(v1) 10 seconds is infinity; reduce to 2 seconds when system is more stable (after we add feature of custom config per test)
const DEFAULT_NODE_COUNT_FOR_ACCEPTANCE = 7
const DEFAULT_ACCEPTANCE_MAX_TX_PER_BLOCK = 10
const DEFAULT_ACCEPTANCE_REQUIRED_QUORUM_PERCENTAGE = 66
const DEFAULT_ACCEPTANCE_VIRTUAL_CHAIN_ID = 42

var DEFAULT_ACCEPTANCE_EMPTY_BLOCK_TIME = 10 * time.Millisecond

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
	virtualChainId           primitives.VirtualChainId
	emptyBlockTime           time.Duration
}

func newHarness() *networkHarnessBuilder {
	n := &networkHarnessBuilder{maxTxPerBlock: DEFAULT_ACCEPTANCE_MAX_TX_PER_BLOCK, requiredQuorumPercentage: DEFAULT_ACCEPTANCE_REQUIRED_QUORUM_PERCENTAGE}

	var algos []consensus.ConsensusAlgoType
	if ENABLE_LEAN_HELIX_IN_ACCEPTANCE_TESTS {
		algos = []consensus.ConsensusAlgoType{consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX, consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS}
	} else {
		algos = []consensus.ConsensusAlgoType{consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS}
	}

	callerFuncName := getCallerFuncName(2)
	if callerFuncName == "getStressTestHarness" {
		callerFuncName = getCallerFuncName(3)
	}

	harness := n.
		WithTestId(callerFuncName).
		WithNumNodes(DEFAULT_NODE_COUNT_FOR_ACCEPTANCE).
		WithConsensusAlgos(algos...).
		WithVirtualChainId(DEFAULT_ACCEPTANCE_VIRTUAL_CHAIN_ID).
		AllowingErrors("ValidateBlockProposal failed.*") // it is acceptable for validation to fail in one or more nodes, as long as f+1 nodes are in agreement on a block and even if they do not, a new leader should eventually be able to reach consensus on the block

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
	logger, testOutput := b.makeLogger(tb, testId)

	supervised.Recover(logger, func() {
		defer testOutput.TestTerminated() // this will suppress test failures from goroutines after test terminates
		// TODO: if we experience flakiness during system shutdown move TestTerminated to be under test.WithContextWithTimeout

		test.WithContextWithTimeout(TEST_TIMEOUT_HARD_LIMIT, func(ctx context.Context) {
			network := newAcceptanceTestNetwork(ctx, logger, consensusAlgo, b.blockChain, b.numNodes, b.maxTxPerBlock, b.requiredQuorumPercentage, b.virtualChainId, b.emptyBlockTime)

			logger.Info("acceptance network created")
			defer printTestIdOnFailure(tb, testId)
			defer dumpStateOnFailure(tb, network)
			defer test.RequireNoUnexpectedErrors(tb, testOutput)

			if b.setupFunc != nil {
				b.setupFunc(ctx, network)
			}

			network.CreateAndStartNodes(ctx, b.numOfNodesToStart)
			logger.Info("acceptance network started")

			logger.Info("acceptance network running test")
			f(tb, ctx, network)
		})

		time.Sleep(10 * time.Millisecond) // give context dependent goroutines time to terminate gracefully
	})
}

func toShortConsensusAlgoStr(algoType consensus.ConsensusAlgoType) string {
	str := algoType.String()
	if len(str) < 20 {
		return str
	}
	return str[20:] // remove the "CONSENSUS_ALGO_TYPE_" prefix
}

func (b *networkHarnessBuilder) makeLogger(tb testing.TB, testId string) (log.Logger, *log.TestOutput) {

	testOutput := log.NewTestOutput(tb, log.NewHumanReadableFormatter())
	for _, pattern := range b.allowedErrors {
		testOutput.AllowErrorsMatching(pattern)
	}

	logger := log.GetLogger(
		log.String("_test", "acceptance"),
		log.String("_test-id", testId)).
		WithOutput(testOutput).
		WithFilters(
			log.IgnoreMessagesMatching("transport message received"),
			log.ExcludeField(memory.LogTag),
		).
		WithFilters(b.logFilters...)
	//WithFilters(log.Or(log.OnlyErrors(), log.OnlyCheckpoints()))

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

func (b *networkHarnessBuilder) WithVirtualChainId(id primitives.VirtualChainId) *networkHarnessBuilder {
	b.virtualChainId = id
	return b
}

func (b *networkHarnessBuilder) WithEmptyBlockTime(emptyBlockTime time.Duration) *networkHarnessBuilder {
	b.emptyBlockTime = emptyBlockTime
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

func getCallerFuncName(skip int) string {
	pc, _, _, _ := runtime.Caller(skip)
	packageAndFuncName := runtime.FuncForPC(pc).Name()
	parts := strings.Split(packageAndFuncName, ".")
	return parts[len(parts)-1]
}

// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package acceptance

import (
	"context"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/memory"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/scribe/log"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const ENABLE_LEAN_HELIX_IN_ACCEPTANCE_TESTS = true
const TEST_TIMEOUT_HARD_LIMIT = 10 * time.Second
const DEFAULT_NODE_COUNT_FOR_ACCEPTANCE = 7
const DEFAULT_ACCEPTANCE_MAX_TX_PER_BLOCK = 10
const DEFAULT_ACCEPTANCE_REQUIRED_QUORUM_PERCENTAGE = 66
const DEFAULT_ACCEPTANCE_VIRTUAL_CHAIN_ID = 42

var DEFAULT_ACCEPTANCE_EMPTY_BLOCK_TIME = 10 * time.Millisecond

type acceptanceNetworkHarness struct {
	sequentialTests          sync.Mutex
	numNodes                 int
	consensusAlgos           []consensus.ConsensusAlgoType
	testId                   string
	setupFunc                func(ctx context.Context, network *Network)
	configOverride           func(config config.OverridableConfig) config.OverridableConfig
	logFilters               []log.Filter
	maxTxPerBlock            uint32
	allowedErrors            []string
	numOfNodesToStart        int
	requiredQuorumPercentage uint32
	blockChain               []*protocol.BlockPairContainer
	virtualChainId           primitives.VirtualChainId
	emptyBlockTime           time.Duration
}

func newHarness() *acceptanceNetworkHarness {
	n := &acceptanceNetworkHarness{maxTxPerBlock: DEFAULT_ACCEPTANCE_MAX_TX_PER_BLOCK, requiredQuorumPercentage: DEFAULT_ACCEPTANCE_REQUIRED_QUORUM_PERCENTAGE}

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
		AllowingErrors(
			"ValidateBlockProposal failed.*", // it is acceptable for validation to fail in one or more nodes, as long as f+1 nodes are in agreement on a block and even if they do not, a new leader should eventually be able to reach consensus on the block
		)

	return harness
}

func (b *acceptanceNetworkHarness) WithLogFilters(filters ...log.Filter) *acceptanceNetworkHarness {
	b.logFilters = filters
	return b
}

func (b *acceptanceNetworkHarness) WithTestId(testId string) *acceptanceNetworkHarness {
	randNum := rand.Intn(1000)
	b.testId = "acc-" + testId + "-" + strconv.FormatInt(time.Now().Unix(), 10) + "-" + strconv.FormatInt(int64(randNum), 10)
	return b
}

func (b *acceptanceNetworkHarness) WithNumNodes(numNodes int) *acceptanceNetworkHarness {
	b.numNodes = numNodes
	return b
}

func (b *acceptanceNetworkHarness) WithConsensusAlgos(algos ...consensus.ConsensusAlgoType) *acceptanceNetworkHarness {
	b.consensusAlgos = algos
	return b
}

// setup runs when all adapters have been created but before the nodes are started
func (b *acceptanceNetworkHarness) WithSetup(f func(ctx context.Context, network *Network)) *acceptanceNetworkHarness {
	b.setupFunc = f
	return b
}

func (b *acceptanceNetworkHarness) WithMaxTxPerBlock(maxTxPerBlock uint32) *acceptanceNetworkHarness {
	b.maxTxPerBlock = maxTxPerBlock
	return b
}

func (b *acceptanceNetworkHarness) AllowingErrors(allowedErrors ...string) *acceptanceNetworkHarness {
	b.allowedErrors = append(b.allowedErrors, allowedErrors...)
	return b
}

func (b *acceptanceNetworkHarness) Start(tb testing.TB, f func(tb testing.TB, ctx context.Context, network *Network)) {
	if b.numOfNodesToStart == 0 {
		b.numOfNodesToStart = b.numNodes
	}

	for _, consensusAlgo := range b.consensusAlgos {
		b.runWithAlgo(tb, consensusAlgo, f)
	}
}

func (b *acceptanceNetworkHarness) runWithAlgo(tb testing.TB, consensusAlgo consensus.ConsensusAlgoType, f func(tb testing.TB, ctx context.Context, network *Network)) {

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

func (b *acceptanceNetworkHarness) runTest(tb testing.TB, consensusAlgo consensus.ConsensusAlgoType, f func(tb testing.TB, ctx context.Context, network *Network)) {
	// acceptance tests are cpu-intensive, so we don't want to run them in parallel
	// as we run subtests, golang will by default run two subtests in parallel
	// we don't want to rely on the -parallel flag, as when running all tests this flag should turn on
	b.sequentialTests.Lock()
	defer b.sequentialTests.Unlock()

	testId := b.testId + "-" + toShortConsensusAlgoStr(consensusAlgo)
	test.WithContext(func(parentCtx context.Context) {
		testOutput := log.NewTestOutput(tb, log.NewHumanReadableFormatter())

		logger := b.makeLogger(testOutput, testId)

		govnr.Recover(logfields.GovnrErrorer(logger), func() {
			ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT_HARD_LIMIT)
			network := newAcceptanceTestNetwork(ctx, logger, consensusAlgo, b.blockChain, b.numNodes, b.maxTxPerBlock, b.requiredQuorumPercentage, b.virtualChainId, b.emptyBlockTime, b.configOverride)
			defer cancel()
			defer testOutput.TestTerminated()

			logger.Info("acceptance network created")
			defer dumpStateOnFailure(tb, network)

			if b.setupFunc != nil {
				b.setupFunc(ctx, network)
			}

			network.CreateAndStartNodes(ctx, b.numOfNodesToStart)
			logger.Info("acceptance network started")

			logger.Info("acceptance network running test")
			f(tb, ctx, network)
			test.RequireNoUnexpectedErrors(tb, testOutput)
			cancel()
			network.WaitUntilShutdown(ctx)

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

func (b *acceptanceNetworkHarness) makeLogger(testOutput *log.TestOutput, testId string) log.Logger {

	for _, pattern := range b.allowedErrors {
		testOutput.AllowErrorsMatching(pattern)
	}

	logger := log.GetLogger().WithTags(
		log.String("_test", "acceptance"),
		log.String("_test-id", testId)).
		WithFilters(
			log.IgnoreMessagesMatching("transport message received"),
			log.ExcludeField(memory.LogTag),
		).
		WithFilters(b.logFilters...)
	//WithFilters(log.Or(log.OnlyErrors(), log.OnlyCheckpoints()))

	return logger
}

func (b *acceptanceNetworkHarness) WithNumRunningNodes(numNodes int) *acceptanceNetworkHarness {
	b.numOfNodesToStart = numNodes
	return b
}

func (b *acceptanceNetworkHarness) WithRequiredQuorumPercentage(percentage int) *acceptanceNetworkHarness {
	b.requiredQuorumPercentage = uint32(percentage)
	return b
}

func (b *acceptanceNetworkHarness) WithInitialBlocks(blocks []*protocol.BlockPairContainer) *acceptanceNetworkHarness {
	b.blockChain = blocks
	return b
}

func (b *acceptanceNetworkHarness) WithVirtualChainId(id primitives.VirtualChainId) *acceptanceNetworkHarness {
	b.virtualChainId = id
	return b
}

func (b *acceptanceNetworkHarness) WithEmptyBlockTime(emptyBlockTime time.Duration) *acceptanceNetworkHarness {
	b.emptyBlockTime = emptyBlockTime
	return b
}

func (b *acceptanceNetworkHarness) WithConfigOverride(f func(cfg config.OverridableConfig) config.OverridableConfig) *acceptanceNetworkHarness {
	b.configOverride = f
	return b
}

func dumpStateOnFailure(tb testing.TB, network *Network) {
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

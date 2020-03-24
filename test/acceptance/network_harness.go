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
	"github.com/orbs-network/orbs-network-go/test/with"
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
const DEFAULT_TEST_TIMEOUT_HARD_LIMIT = 10 * time.Second
const DEFAULT_NODE_COUNT_FOR_ACCEPTANCE = 7
const DEFAULT_ACCEPTANCE_MAX_TX_PER_BLOCK = 10
const DEFAULT_ACCEPTANCE_REQUIRED_QUORUM_PERCENTAGE = 66
const DEFAULT_ACCEPTANCE_VIRTUAL_CHAIN_ID = 42

var DEFAULT_ACCEPTANCE_EMPTY_BLOCK_TIME = 10 * time.Millisecond

type networkHarness struct {
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
	testTimeout              time.Duration
	mangementUpdateTime		 time.Duration
}

func NewHarness() *networkHarness {
	n := &networkHarness{maxTxPerBlock: DEFAULT_ACCEPTANCE_MAX_TX_PER_BLOCK, requiredQuorumPercentage: DEFAULT_ACCEPTANCE_REQUIRED_QUORUM_PERCENTAGE, mangementUpdateTime: 0}

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
		WithTestTimeout(DEFAULT_TEST_TIMEOUT_HARD_LIMIT).
		WithNumNodes(DEFAULT_NODE_COUNT_FOR_ACCEPTANCE).
		WithConsensusAlgos(algos...).
		WithVirtualChainId(DEFAULT_ACCEPTANCE_VIRTUAL_CHAIN_ID).
		AllowingErrors(
			"ValidateBlockProposal failed.*", // it is acceptable for validation to fail in one or more nodes, as long as f+1 nodes are in agreement on a block and even if they do not, a new leader should eventually be able to reach consensus on the block
		)

	return harness
}

func (b *networkHarness) WithLogFilters(filters ...log.Filter) *networkHarness {
	b.logFilters = filters
	return b
}

func (b *networkHarness) WithTestId(testId string) *networkHarness {
	randNum := rand.Intn(1000)
	b.testId = "acc-" + testId + "-" + strconv.FormatInt(time.Now().Unix(), 10) + "-" + strconv.FormatInt(int64(randNum), 10)
	return b
}

func (b *networkHarness) WithTestTimeout(timeout time.Duration) *networkHarness {
	b.testTimeout = timeout
	return b
}

func (b *networkHarness) WithNumNodes(numNodes int) *networkHarness {
	b.numNodes = numNodes
	return b
}

func (b *networkHarness) WithConsensusAlgos(algos ...consensus.ConsensusAlgoType) *networkHarness {
	b.consensusAlgos = algos
	return b
}

// setup runs when all adapters have been created but before the nodes are started
func (b *networkHarness) WithSetup(f func(ctx context.Context, network *Network)) *networkHarness {
	b.setupFunc = f
	return b
}

func (b *networkHarness) WithMaxTxPerBlock(maxTxPerBlock uint32) *networkHarness {
	b.maxTxPerBlock = maxTxPerBlock
	return b
}

func (b *networkHarness) WithManagementUpdateInterval(updateTime time.Duration) *networkHarness {
	b.mangementUpdateTime = updateTime
	return b
}

func (b *networkHarness) AllowingErrors(allowedErrors ...string) *networkHarness {
	b.allowedErrors = append(b.allowedErrors, allowedErrors...)
	return b
}

func (b *networkHarness) Start(tb testing.TB, f func(tb testing.TB, ctx context.Context, network *Network)) {
	if b.numOfNodesToStart == 0 {
		b.numOfNodesToStart = b.numNodes
	}

	for _, consensusAlgo := range b.consensusAlgos {
		b.runWithAlgo(tb, consensusAlgo, f)
	}
}

func (b *networkHarness) runWithAlgo(tb testing.TB, consensusAlgo consensus.ConsensusAlgoType, f func(tb testing.TB, ctx context.Context, network *Network)) {

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

func startHeartbeat(ctx context.Context, logger log.Logger) *govnr.ForeverHandle {
	const heartbeatInterval = 100 * time.Millisecond
	return govnr.Forever(ctx, "heartbeat", logfields.GovnrErrorer(logger), func() {
		logger.Info("heartbeat")
		time.Sleep(heartbeatInterval)
	})
}

func (b *networkHarness) runTest(tb testing.TB, consensusAlgo consensus.ConsensusAlgoType, f func(tb testing.TB, ctx context.Context, network *Network)) {
	// acceptance tests are cpu-intensive, so we don't want to run them in parallel
	// as we run subtests, golang will by default run two subtests in parallel
	// we don't want to rely on the -parallel flag, as when running all tests this flag should turn on
	b.sequentialTests.Lock()
	defer b.sequentialTests.Unlock()

	testId := b.testId + "-" + toShortConsensusAlgoStr(consensusAlgo)
	with.Concurrency(tb, func(parentCtx context.Context, parentHarness *with.ConcurrencyHarness) {
		logger := b.makeLogger(parentHarness, testId)

		govnr.Recover(logfields.GovnrErrorer(logger), func() {
			ctx, cancel := context.WithTimeout(context.Background(), b.testTimeout)
			defer cancel()

			network := newAcceptanceTestNetwork(ctx, logger, consensusAlgo, b.blockChain, b.numNodes, b.maxTxPerBlock, b.requiredQuorumPercentage, b.virtualChainId, b.emptyBlockTime, b.mangementUpdateTime, b.configOverride)
			parentHarness.Supervise(startHeartbeat(ctx, logger))
			parentHarness.Supervise(network)
			defer dumpStateOnFailure(tb, network)
			logger.Info("acceptance network created")

			if b.setupFunc != nil {
				b.setupFunc(ctx, network)
			}

			network.CreateAndStartNodes(ctx, b.numOfNodesToStart)
			logger.Info("acceptance network started, running tests")
			f(tb, ctx, network)
			logger.Info("test completed, network will shut down")

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

func (b *networkHarness) makeLogger(parentHarness *with.ConcurrencyHarness, testId string) log.Logger {

	for _, pattern := range b.allowedErrors {
		parentHarness.AllowErrorsMatching(pattern)
	}

	logger := parentHarness.Logger.WithTags(
		log.String("_test", "acceptance"),
		log.String("_test-id", testId)).
		WithFilters(
			//log.IgnoreMessagesMatching("transport message received"),
			log.ExcludeField(memory.LogTag),
		).
		WithFilters(b.logFilters...)
	//WithFilters(log.Or(log.OnlyErrors(), log.OnlyCheckpoints()))

	return logger
}
func (b *networkHarness) WithNumRunningNodes(numNodes int) *networkHarness {
	b.numOfNodesToStart = numNodes
	return b
}

func (b *networkHarness) WithRequiredQuorumPercentage(percentage int) *networkHarness {
	b.requiredQuorumPercentage = uint32(percentage)
	return b
}

func (b *networkHarness) WithInitialBlocks(blocks []*protocol.BlockPairContainer) *networkHarness {
	b.blockChain = blocks
	return b
}

func (b *networkHarness) WithVirtualChainId(id primitives.VirtualChainId) *networkHarness {
	b.virtualChainId = id
	return b
}

func (b *networkHarness) WithEmptyBlockTime(emptyBlockTime time.Duration) *networkHarness {
	b.emptyBlockTime = emptyBlockTime
	return b
}

func (b *networkHarness) WithConfigOverride(f func(cfg config.OverridableConfig) config.OverridableConfig) *networkHarness {
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

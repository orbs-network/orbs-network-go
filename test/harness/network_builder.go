package harness

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

type acceptanceTestNetworkBuilder struct {
	t              *testing.T
	numNodes       uint32
	consensusAlgos []consensus.ConsensusAlgoType
	testId         string
	setupFunc      func(network InProcessNetwork)
	logFilters     []log.Filter
}

func Network(t *testing.T) *acceptanceTestNetworkBuilder {
	n := &acceptanceTestNetworkBuilder{t: t}

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

func (b *acceptanceTestNetworkBuilder) WithNumNodes(numNodes uint32) *acceptanceTestNetworkBuilder {
	b.numNodes = numNodes
	return b
}

func (b *acceptanceTestNetworkBuilder) WithConsensusAlgos(algos ...consensus.ConsensusAlgoType) *acceptanceTestNetworkBuilder {
	b.consensusAlgos = algos
	return b
}

// setup runs when all adapters have been created but before the nodes are started
func (b *acceptanceTestNetworkBuilder) WithSetup(f func(network InProcessNetwork)) *acceptanceTestNetworkBuilder {
	b.setupFunc = f
	return b
}

func (b *acceptanceTestNetworkBuilder) Start(f func(network InProcessNetwork)) {
	for _, consensusAlgo := range b.consensusAlgos {

		// start test
		test.WithContext(func(ctx context.Context) {
			testId := b.testId + "-" + consensusAlgo.String()
			network := NewAcceptanceTestNetwork(b.numNodes, b.logFilters, consensusAlgo, testId)

			defer printTestIdOnFailure(b.t, testId)
			defer dumpStateOnFailure(b.t, network)

			if b.setupFunc != nil {
				b.setupFunc(network)
			}

			network.StartNodes(ctx)

			f(network)
		})
		// end test

		time.Sleep(5 * time.Millisecond) // give context dependent goroutines 5 ms to terminate gracefully
	}
}

func printTestIdOnFailure(t *testing.T, testId string) {
	if t.Failed() {
		fmt.Println("FAIL search snippet: grep _test-id="+testId, "test.out")
	}
}

func dumpStateOnFailure(t *testing.T, network InProcessNetwork) {
	if t.Failed() {
		network.DumpState()
	}
}

func getCallerFuncName() string {
	pc, _, _, _ := runtime.Caller(2)
	packageAndFuncName := runtime.FuncForPC(pc).Name()
	parts := strings.Split(packageAndFuncName, ".")
	return parts[len(parts)-1]
}

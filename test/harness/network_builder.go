package harness

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"math/rand"
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
	setupFunc      func(network AcceptanceTestNetwork)
}

func Network(t *testing.T) *acceptanceTestNetworkBuilder {
	n := &acceptanceTestNetworkBuilder{t: t}

	return n.
		WithTestId(getCallerFuncName()).
		WithNumNodes(2).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS)
}

func (b *acceptanceTestNetworkBuilder) WithTestId(testId string) *acceptanceTestNetworkBuilder {
	b.testId = "acceptance-" + testId + "-" + strconv.FormatUint(rand.Uint64(), 10)
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
func (b *acceptanceTestNetworkBuilder) WithSetup(f func(network AcceptanceTestNetwork)) *acceptanceTestNetworkBuilder {
	b.setupFunc = f
	return b
}

func (b *acceptanceTestNetworkBuilder) Start(f func(network AcceptanceTestNetwork)) {
	for _, consensusAlgo := range b.consensusAlgos {

		// start test
		test.WithContext(func(ctx context.Context) {
			testId := b.testId + "-" + consensusAlgo.String()
			network := NewAcceptanceTestNetwork(b.numNodes, consensusAlgo, testId)

			defer printTestIdOnFailure(b.t, testId)
			defer dumpStateOnFailure(b.t, network)

			if b.setupFunc != nil {
				b.setupFunc(network)
			}

			network.StartNodes(ctx)

			f(network)
		})
		// end test

		time.Sleep(1 * time.Millisecond) // give context dependent goroutines 1 ms to terminate gracefully
	}
}

func printTestIdOnFailure(t *testing.T, testId string) {
	if t.Failed() {
		fmt.Println("FAIL search snippet: grep _test-id="+testId, "test.out")
	}
}

func dumpStateOnFailure(t *testing.T, network AcceptanceTestNetwork) {
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

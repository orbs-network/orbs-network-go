package harness

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type canFail interface {
	Failed() bool
	Fatal(args ...interface{})
}

type acceptanceTestNetworkBuilder struct {
	f              canFail
	numNodes       uint32
	consensusAlgos []consensus.ConsensusAlgoType
	testId         string
	setupFunc      func(ctx context.Context, network InProcessTestNetwork)
	logFilters     []log.Filter
	maxTxPerBlock  uint32
	allowedErrors  []string
}

func Network(f canFail) *acceptanceTestNetworkBuilder {
	n := &acceptanceTestNetworkBuilder{f: f, maxTxPerBlock: 30}

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
func (b *acceptanceTestNetworkBuilder) WithSetup(f func(ctx context.Context, network InProcessTestNetwork)) *acceptanceTestNetworkBuilder {
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

func (b *acceptanceTestNetworkBuilder) Start(f func(ctx context.Context, network InProcessTestNetwork)) {
	for _, consensusAlgo := range b.consensusAlgos {

		// start test
		test.WithContext(func(ctx context.Context) {
			testId := b.testId + "-" + consensusAlgo.String()
			logger := b.makeLogger(testId)
			network := NewAcceptanceTestNetwork(b.numNodes, logger, consensusAlgo, b.maxTxPerBlock)

			defer printTestIdOnFailure(b.f, testId)
			defer dumpStateOnFailure(b.f, network)
			defer func() {
				if logger.HasErrors() {
					b.f.Fatal("Encountered unexpected errors:\n\t", strings.Join(logger.GetUnexpectedErrors(), "\n\t"))

				}
			}()

			if b.setupFunc != nil {
				b.setupFunc(ctx, network)
			}

			network.StartNodes(ctx)

			f(ctx, network)
		})
		// end test

		time.Sleep(5 * time.Millisecond) // give context dependent goroutines 5 ms to terminate gracefully
	}
}

func (b *acceptanceTestNetworkBuilder) makeLogger(testId string) *log.ErrorRecordingLogger {
	var output io.Writer
	output = os.Stdout

	if os.Getenv("NO_LOG_STDOUT") == "true" {
		logFile, err := os.OpenFile(config.GetProjectSourceRootPath()+"/logs/acceptance/"+testId+".log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}

		output = logFile
	}

	logger := log.GetLogger(
		log.String("_test", "acceptance"),
		log.String("_branch", os.Getenv("GIT_BRANCH")),
		log.String("_commit", os.Getenv("GIT_COMMIT")),
		log.String("_test-id", testId),
	).
		WithOutput(log.NewOutput(output).WithFormatter(log.NewJsonFormatter())).
		WithFilters(b.logFilters...).
		WithFilters(log.Or(log.OnlyErrors(), log.OnlyCheckpoints(), log.OnlyMetrics()))

	return log.NewErrorRecordingLogger(logger, b.allowedErrors)
}

func printTestIdOnFailure(f canFail, testId string) {
	if f.Failed() {
		fmt.Println("FAIL search snippet: grep _test-id="+testId, "test.out")
	}
}

func dumpStateOnFailure(f canFail, network InProcessTestNetwork) {
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

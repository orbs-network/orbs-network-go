//+build goroutineleak

package acceptance

import (
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/stretchr/testify/require"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"
)

// this test should not run in parallel with any other test (even package parallel) since it's examining shared global system state (num goroutines)
// if another test is running, the other test may create goroutines which we may mistake as leaks because the numbers won't add up
// therefore, this file is marked on top with a build flag ("goroutineleak") meaning without this flag it won't build or run
// to run this test, add to the go command "-tags goroutineleak", this is done in test.sh while making sure it's the only test running
func TestGoroutineLeaks_OnSystemShutdown_LeanHelix(t *testing.T) {
	testGoroutineLeaksWithAlgo(t, consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX, rand.NewControlledRand(t))
}

func TestGoroutineLeaks_OnSystemShutdown_BenchmarkConsensus(t *testing.T) {
	testGoroutineLeaksWithAlgo(t, consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS, rand.NewControlledRand(t))
}

func testGoroutineLeaksWithAlgo(t *testing.T, algo consensus.ConsensusAlgoType, rnd *rand.ControlledRand) {
	before, _ := os.Create("/tmp/gorou-shutdown-before.out")
	defer before.Close()
	after, _ := os.Create("/tmp/gorou-shutdown-after.out")
	defer after.Close()
	numGoroutineBefore := runtime.NumGoroutine()
	pprof.Lookup("goroutine").WriteTo(before, 1)
	runHappyFlowWithConsensusAlgo(t, algo)
	time.Sleep(100 * time.Millisecond)
	// give goroutines time to terminate
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	// give goroutines time to terminate
	numGoroutineAfter := runtime.NumGoroutine()
	pprof.Lookup("goroutine").WriteTo(after, 1)
	require.Equal(t, numGoroutineBefore, numGoroutineAfter, "number of goroutines should be equal, to see stack traces of the leaks, compare the files: /tmp/gorou-shutdown-before.out /tmp/gorou-shutdown-after.out")
}

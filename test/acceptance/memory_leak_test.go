//+build memoryleak

package acceptance

import (
	"github.com/stretchr/testify/require"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"
)

// this test should not run in parallel with any other test (even package parallel) since it's examining shared global system state (num goroutines)
// if another test is running, the other test may create goroutines which we may mistake as leaks because the numbers won't add up
// therefore, this file is marked on top with a build flag ("memoryleak") meaning without this flag it won't build or run
// to run this test, add to the go command "-tags memoryleak", this is done in test.sh while making sure it's the only test running
func TestMemoryLeaks_OnSystemShutdown(t *testing.T) {

	before, _ := os.Create("/tmp/mem-shutdown-before.prof")
	defer before.Close()
	after, _ := os.Create("/tmp/mem-shutdown-after.prof")
	defer after.Close()

	t.Run("TestCreateGazillionTransactionsWhileTransportIsDuplicatingRandomMessages", TestCreateGazillionTransactionsWhileTransportIsDuplicatingRandomMessages)
	t.Run("TestCreateGazillionTransactionsWhileTransportIsDroppingRandomMessages", TestCreateGazillionTransactionsWhileTransportIsDroppingRandomMessages)
	t.Run("TestCreateGazillionTransactionsWhileTransportIsDelayingRandomMessages", TestCreateGazillionTransactionsWhileTransportIsDelayingRandomMessages)

	runtime.GC()
	runtime.GC()
	runtime.GC()
	runtime.GC()
	time.Sleep(50 * time.Millisecond)

	memUsageBefore := getMemUsage()
	pprof.WriteHeapProfile(before)

	for i := 0; i < 10; i++ {
		t.Run("TestCreateGazillionTransactionsWhileTransportIsDuplicatingRandomMessages", TestCreateGazillionTransactionsWhileTransportIsDuplicatingRandomMessages)
		t.Run("TestCreateGazillionTransactionsWhileTransportIsDroppingRandomMessages", TestCreateGazillionTransactionsWhileTransportIsDroppingRandomMessages)
		t.Run("TestCreateGazillionTransactionsWhileTransportIsDelayingRandomMessages", TestCreateGazillionTransactionsWhileTransportIsDelayingRandomMessages)
	}

	runtime.GC()
	runtime.GC()
	runtime.GC()
	runtime.GC()
	time.Sleep(50 * time.Millisecond)

	memUsageAfter := getMemUsage()
	pprof.WriteHeapProfile(after)

	require.InDelta(t, memUsageAfter, memUsageBefore, 0.1*float64(memUsageBefore), "added memory should be around than 10%, compare /tmp/mem-shutdown-before.prof and /tmp/mem-shutdown-after.prof to see memory consumers")
}

func getMemUsage() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return m.Alloc
}

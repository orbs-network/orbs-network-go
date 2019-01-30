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
	t.Skip("skip temporarily")

	before, _ := os.Create("/tmp/mem-shutdown-before.prof")
	defer before.Close()
	after, _ := os.Create("/tmp/mem-shutdown-after.prof")
	defer after.Close()

	t.Run("TestCreateGazillionTransactionsWhileTransportIsDuplicatingRandomMessages", TestCreateGazillionTransactionsWhileTransportIsDuplicatingRandomMessages)
	t.Run("TestCreateGazillionTransactionsWhileTransportIsDroppingRandomMessages", TestCreateGazillionTransactionsWhileTransportIsDroppingRandomMessages)
	t.Run("TestCreateGazillionTransactionsWhileTransportIsDelayingRandomMessages", TestCreateGazillionTransactionsWhileTransportIsDelayingRandomMessages)

	time.Sleep(100 * time.Millisecond)
	runtime.GC()
	runtime.GC()
	runtime.GC()
	runtime.GC()

	memUsageBeforeBytes := getMemUsageBytes()
	pprof.WriteHeapProfile(before)

	for i := 0; i < 20; i++ {
		t.Run("TestCreateGazillionTransactionsWhileTransportIsDuplicatingRandomMessages", TestCreateGazillionTransactionsWhileTransportIsDuplicatingRandomMessages)
		t.Run("TestCreateGazillionTransactionsWhileTransportIsDroppingRandomMessages", TestCreateGazillionTransactionsWhileTransportIsDroppingRandomMessages)
		t.Run("TestCreateGazillionTransactionsWhileTransportIsDelayingRandomMessages", TestCreateGazillionTransactionsWhileTransportIsDelayingRandomMessages)
	}

	time.Sleep(100 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	runtime.GC()

	memUsageAfterBytes := getMemUsageBytes()
	pprof.WriteHeapProfile(after)

	deltaMemBytes := memUsageAfterBytes - memUsageBeforeBytes
	allowedMemIncreaseCalculatedFromMemBefore := uint64(0.1 * float64(memUsageBeforeBytes))
	allowedMemIncreaseInAbsoluteBytes := uint64(512 * 1024)

	require.Conditionf(t, func() bool {
		return deltaMemBytes < allowedMemIncreaseCalculatedFromMemBefore || deltaMemBytes < allowedMemIncreaseInAbsoluteBytes
	}, "Heap size after GC is too large. Pre-run: %d bytes, post-run: %d bytes, added %d bytes. This is more than 10%% of initial memory and more than the allowed addition of %d bytes. Compare /tmp/mem-shutdown-before.prof and /tmp/mem-shutdown-after.prof to see memory consumers",
		memUsageBeforeBytes, memUsageAfterBytes, deltaMemBytes, allowedMemIncreaseInAbsoluteBytes)
}

func getMemUsageBytes() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return m.Alloc
}

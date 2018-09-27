//+build goroutineleak

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
// therefore, this file is marked on top with a build flag ("goroutineleak") meaning without this flag it won't build or run
// to run this test, add to the go command "-tags goroutineleak", this is done in test.sh while making sure it's the only test running
func TestGoroutineLeaks_OnSystemShutdown(t *testing.T) {

	before, _ := os.Create("/tmp/gorou-shutdown-before.out")
	defer before.Close()
	after, _ := os.Create("/tmp/gorou-shutdown-after.out")
	defer after.Close()

	numGoroutineBefore := runtime.NumGoroutine()
	pprof.Lookup("goroutine").WriteTo(before, 1)

	t.Run("TestCreateGazillionTransactionsWhileTransportIsDuplicatingRandomMessages", TestCreateGazillionTransactionsWhileTransportIsDuplicatingRandomMessages)
	t.Run("TestCreateGazillionTransactionsWhileTransportIsDroppingRandomMessages", TestCreateGazillionTransactionsWhileTransportIsDroppingRandomMessages)
	t.Run("TestCreateGazillionTransactionsWhileTransportIsDelayingRandomMessages", TestCreateGazillionTransactionsWhileTransportIsDelayingRandomMessages)

	runtime.GC()
	time.Sleep(50 * time.Millisecond) // give goroutines time to terminate

	numGoroutineAfter := runtime.NumGoroutine()
	pprof.Lookup("goroutine").WriteTo(after, 1)

	require.Equal(t, numGoroutineBefore, numGoroutineAfter, "number of goroutines should be equal, compare /tmp/gorou-shutdown-before.out and /tmp/gorou-shutdown-after.out to see stack traces of the leaks")
}

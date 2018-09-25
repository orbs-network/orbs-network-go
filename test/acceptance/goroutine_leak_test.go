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

// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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

	runtime.GC()
	time.Sleep(100 * time.Millisecond) // give goroutines time to terminate

	numGoroutineBefore := runtime.NumGoroutine()
	pprof.Lookup("goroutine").WriteTo(before, 1)

	t.Run("TestGazillionTxWhileDuplicatingMessages", TestGazillionTxWhileDuplicatingMessages)
	t.Run("TestGazillionTxWhileDroppingMessages", TestGazillionTxWhileDroppingMessages)
	t.Run("TestGazillionTxWhileDelayingMessages", TestGazillionTxWhileDelayingMessages)

	time.Sleep(100 * time.Millisecond) // give goroutines time to terminate
	runtime.GC()
	time.Sleep(100 * time.Millisecond) // give goroutines time to terminate

	numGoroutineAfter := runtime.NumGoroutine()
	pprof.Lookup("goroutine").WriteTo(after, 1)

	require.Equal(t, numGoroutineBefore, numGoroutineAfter, "number of goroutines should be equal, compare /tmp/gorou-shutdown-before.out and /tmp/gorou-shutdown-after.out to see stack traces of the leaks")
}

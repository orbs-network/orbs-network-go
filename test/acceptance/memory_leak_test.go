// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

// +build memoryleak

package acceptance

import (
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
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
	t.Skip("This test is not useful, we only care about memory leaks during runtime and this is covered by the monitoring project 'marvin': https://github.com/orbs-network/marvin")
	dir := os.Getenv("PPROF_DIR")
	absPath, err := filepath.Abs("../../" + dir)

	require.NoError(t, err)

	t.Log("Will save profiling results in " + absPath)

	beforeProf := absPath + "/mem-shutdown-before.prof"
	afterProf := absPath + "/mem-shutdown-after.prof"
	before, err := os.Create(beforeProf)
	defer before.Close()
	require.NoError(t, err)
	after, err := os.Create(afterProf)
	defer after.Close()
	require.NoError(t, err)

	t.Run("TestGazillionTxWhileDuplicatingMessages", TestGazillionTxWhileDuplicatingMessages)
	t.Run("TestGazillionTxWhileDroppingMessages", TestGazillionTxWhileDroppingMessages)
	t.Run("TestGazillionTxWhileDelayingMessages", TestGazillionTxWhileDelayingMessages)

	time.Sleep(100 * time.Millisecond)
	runtime.GC()
	runtime.GC()
	runtime.GC()
	runtime.GC()

	memUsageBeforeBytes := getMemUsageBytes()
	pprof.WriteHeapProfile(before)

	for i := 0; i < 20; i++ {
		t.Run("TestGazillionTxWhileDuplicatingMessages", TestGazillionTxWhileDuplicatingMessages)
		t.Run("TestGazillionTxWhileDroppingMessages", TestGazillionTxWhileDroppingMessages)
		t.Run("TestGazillionTxWhileDelayingMessages", TestGazillionTxWhileDelayingMessages)
	}

	sleepAndGC(t)
	sleepAndGC(t)
	sleepAndGC(t)
	sleepAndGC(t)

	memUsageAfterBytes := getMemUsageBytes()
	pprof.WriteHeapProfile(after)

	if memUsageAfterBytes < memUsageBeforeBytes {
		return // its okay if the after is less in memory, no leak (and the rest of the math will overflow)
	}

	deltaMemBytes := memUsageAfterBytes - memUsageBeforeBytes
	allowedMemIncreaseCalculatedFromMemBefore := uint64(0.1 * float64(memUsageBeforeBytes))
	allowedMemIncreaseInAbsoluteBytes := uint64(1 * 1024 * 1024) // 1MB

	require.Conditionf(t, func() bool {
		return deltaMemBytes < allowedMemIncreaseCalculatedFromMemBefore || deltaMemBytes < allowedMemIncreaseInAbsoluteBytes
	}, "Heap size after GC is too large. Pre-run: %d bytes, post-run: %d bytes, added %d bytes. This is more than 10%% of initial memory and more than the allowed addition of %d bytes. Compare %s and %s to see memory consumers",
		memUsageBeforeBytes, memUsageAfterBytes, deltaMemBytes, allowedMemIncreaseInAbsoluteBytes, beforeProf, afterProf)
}

func TestMemoryLeaks_DuringRuntime(t *testing.T) {
	t.Skip("This test is incorrect - should modify it to take memory samples within TestGazillion instead, see https://github.com/orbs-network/orbs-network-go/issues/1346")
	dir := os.Getenv("PPROF_DIR")
	absPath, err := filepath.Abs("../../" + dir)

	require.NoError(t, err)

	t.Log("Will save profiling results in " + absPath)

	beforeProf := absPath + "/mem-runtime-before.prof"
	afterProf := absPath + "/mem-runtime-after.prof"
	before, err := os.Create(beforeProf)
	defer before.Close()
	require.NoError(t, err)
	after, err := os.Create(afterProf)
	defer after.Close()
	require.NoError(t, err)

	var warmups = 2
	var repetitions = 5
	var memUsageBeforeBytes uint64
	var memUsageAfterBytes uint64
	for i := 0; i < warmups; i++ {
		t.Run("TestGazillionTxWhileDuplicatingMessages", TestGazillionTxWhileDuplicatingMessages)
		t.Run("TestGazillionTxWhileDroppingMessages", TestGazillionTxWhileDroppingMessages)
		t.Run("TestGazillionTxWhileDelayingMessages", TestGazillionTxWhileDelayingMessages)
		if i == warmups-1 {
			memUsageBeforeBytes = getMemUsageBytes()
			pprof.WriteHeapProfile(before)
		}
	}

	for i := 0; i < repetitions; i++ {
		t.Run("TestGazillionTxWhileDuplicatingMessages", TestGazillionTxWhileDuplicatingMessages)
		t.Run("TestGazillionTxWhileDroppingMessages", TestGazillionTxWhileDroppingMessages)
		t.Run("TestGazillionTxWhileDelayingMessages", TestGazillionTxWhileDelayingMessages)
		if i == repetitions-1 {
			memUsageAfterBytes = getMemUsageBytes()
			pprof.WriteHeapProfile(after)
		}
	}
	if memUsageAfterBytes < memUsageBeforeBytes {
		return // its okay if the after is less in memory, no leak (and the rest of the math will overflow)
	}

	deltaMemBytes := memUsageAfterBytes - memUsageBeforeBytes
	allowedMemIncreaseCalculatedFromMemBefore := uint64(0.1 * float64(memUsageBeforeBytes))
	allowedMemIncreaseInAbsoluteBytes := uint64(1 * 1024 * 1024) // 1MB
	t.Logf("Pre-run: %d bytes, during-run: %d bytes, added %d bytes",
		memUsageBeforeBytes, memUsageAfterBytes, deltaMemBytes)

	require.Conditionf(t, func() bool {
		return deltaMemBytes < allowedMemIncreaseCalculatedFromMemBefore || deltaMemBytes < allowedMemIncreaseInAbsoluteBytes
	}, "Heap size increased too much during runtime.\nPre-run: %d bytes, during-run: %d bytes, added %d bytes. \nThis is more than 10%% of initial memory and more than the allowed addition of %d bytes. \nCompare %s and %s to see memory consumers",
		memUsageBeforeBytes, memUsageAfterBytes, deltaMemBytes, allowedMemIncreaseInAbsoluteBytes, beforeProf, afterProf)
}

func sleepAndGC(t testing.TB) {
	before := getMemUsageBytes()
	time.Sleep(400 * time.Millisecond)
	runtime.GC()
	after := getMemUsageBytes()
	t.Logf("Memory usage before GC %d bytes, after %d bytes, delta %d bytes", before, after, before-after)
}

func getMemUsageBytes() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return m.Alloc
}

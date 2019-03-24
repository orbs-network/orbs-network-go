// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"math/rand"
	"runtime"
	"sync"
	"time"
)

var (
	MIN_REST_DURATION  = 1 * time.Millisecond
	MAX_REST_DURATION  = 5 * time.Millisecond
	MIN_BURST_DURATION = 1 * time.Millisecond
	MAX_BURST_DURATION = 10 * time.Millisecond
)

var once sync.Once

// creates ongoing random bursts of cpu noise (all cores together) to make goroutine scheduling erratic in -count 100 flakiness tests
func StartCpuSchedulingJitter() {
	once.Do(func() {
		go generateCpuNoiseRunLoop()
	})
}

func generateCpuNoiseRunLoop() {
	r := rand.New(rand.NewSource(int64(17)))
	for {

		restDuration := randDurationInRange(r, MIN_REST_DURATION, MAX_REST_DURATION)
		burstDuration := randDurationInRange(r, MIN_BURST_DURATION, MAX_BURST_DURATION)

		time.Sleep(restDuration)

		cpuNoiseBurst(burstDuration, runtime.GOMAXPROCS(0))
	}
}

func randDurationInRange(r *rand.Rand, min time.Duration, max time.Duration) time.Duration {
	return min + time.Duration(r.Int63n(int64(max-min)))
}

func cpuNoiseBurst(burstDuration time.Duration, numCores int) {
	var wg sync.WaitGroup
	burstDeadline := time.Now().Add(burstDuration)
	for i := 0; i < numCores; i++ {
		wg.Add(1)
		go cpuNoiseBurstPerCore(burstDeadline, &wg)
	}
	wg.Wait()
}

func cpuNoiseBurstPerCore(burstDeadline time.Time, wg *sync.WaitGroup) {
	for time.Now().Before(burstDeadline) {
		// busy wait
	}
	wg.Done()
}

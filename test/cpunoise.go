package test

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

var (
	MIN_REST_DURATION      = 2 * time.Millisecond
	MAX_REST_DURATION      = 5 * time.Millisecond
	MIN_BURST_DURATION     = 100 * time.Microsecond
	MAX_BURST_DURATION     = 500 * time.Microsecond
	MAX_ALLOWED_STARVATION = 15 * time.Millisecond
)

var once sync.Once

// creates ongoing random bursts of cpu noise (all cores together) to make goroutine scheduling erratic in -count 100 flakiness tests
func StartCpuSchedulingJitter() {
	once.Do(func() {
		runtime.GOMAXPROCS(8)
		// go generateCpuNoiseRunLoop()
		go verifyNoStarvationRunLoop()
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

func verifyNoStarvationRunLoop() {
	pollInterval := 2 * time.Millisecond
	lastScheduled := time.Now()
	for {

		time.Sleep(pollInterval)

		now := time.Now()
		starved := now.Sub(lastScheduled) - pollInterval
		if starved > MAX_ALLOWED_STARVATION {
			panic(fmt.Sprintf("cpunoise is causing goroutine starvation! configure it to be less aggressive\nstarved for %d milliseconds, with %d active cores, total %d goroutines", starved/1000000, runtime.GOMAXPROCS(0), runtime.NumGoroutine()))
		}
		lastScheduled = now

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

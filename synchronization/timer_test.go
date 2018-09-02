package synchronization_test

import (
	"github.com/orbs-network/orbs-network-go/synchronization"
	"testing"
	"time"
)

func TestTimer_Reset(t *testing.T) {
	start := time.Now()

	timer := synchronization.NewTimer(10 * time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	timer.Reset(10 * time.Millisecond)

	// See issue: https://github.com/golang/go/issues/11513 , this timer should solve it, the reset above should 'reset' the channel as well
	<-timer.C

	runtime := time.Since(start).Seconds()

	if runtime < 0.03 {
		t.Errorf("took ~%v milliseconds, should be ~30 milliseconds\n", runtime*1000)
	}
}

func TestTimer_Stop(t *testing.T) {
	timer := synchronization.NewTimer(10 * time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	timer.Stop()
	releaseC := make(chan bool, 1)
	time.AfterFunc(10*time.Millisecond, func() { releaseC <- true })

	select {
	case <-timer.C:
		t.Error("timer pinged, but it was stopped already")
	case <-releaseC:
	}
}

func TestTimer_GetTimer(t *testing.T) {
	x := synchronization.NewTimer(time.Second)
	if x == nil {
		t.Error("cannot create new timer")
	}

	timer := x.GetTimer()
	if timer == nil {
		t.Error("timer inner object was not created, something is really wrong")
	}
}

func TestValidTimerStopOp(t *testing.T) {
	// testing that a timer stopped before it was expired does not later ping
	timer := synchronization.NewTimer(5 * time.Millisecond)
	timer.Stop()
	releaseC := make(chan bool, 1)
	time.AfterFunc(10*time.Millisecond, func() { releaseC <- true })

	select {
	case <-timer.C:
		t.Error("timer pinged, but it was stopped already")
	case <-releaseC:
	}
}

func TestValidTimerResetOp(t *testing.T) {
	// testing that a timer reset before it was expired does not ping too early
	start := time.Now()

	timer := synchronization.NewTimer(5 * time.Millisecond)
	timer.Reset(10 * time.Millisecond)
	<-timer.C

	runtime := time.Since(start).Seconds()

	if runtime < 0.01 {
		t.Errorf("took ~%v milliseconds, should be ~10 milliseconds\n", runtime*1000)
	}
}

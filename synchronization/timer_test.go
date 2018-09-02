package synchronization_test

import (
	"github.com/orbs-network/orbs-network-go/synchronization"
	"testing"
	"time"
)

func TestNewTimer(t *testing.T) {
	x := synchronization.NewTimer(time.Second)
	if x == nil {
		t.Fail()
	}
}

func TestReset(t *testing.T) {
	start := time.Now()

	timer := synchronization.NewTimer(100 * time.Millisecond)
	time.Sleep(200 * time.Millisecond)
	timer.Reset(100 * time.Millisecond)

	// See issue: https://github.com/golang/go/issues/11513 , this timer should solve it, the reset above should 'reset' the channel as well
	<-timer.C

	runtime := time.Since(start).Seconds()

	if runtime < 0.3 {
		t.Fatalf("took ~%v milliseconds, should be ~300 milliseconds\n", runtime*1000)
	}
}

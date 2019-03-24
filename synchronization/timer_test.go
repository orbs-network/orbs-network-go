// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package synchronization_test

import (
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestTimer_Reset(t *testing.T) {
	start := time.Now()

	// replacing synchronization to time (using the standard timer) will break the test and show the issue
	timer := synchronization.NewTimer(5 * time.Millisecond)
	time.Sleep(7 * time.Millisecond)
	timer.Reset(2 * time.Millisecond)

	// See issue: https://github.com/golang/go/issues/11513 , this timer should solve it, the reset above should 'reset' the channel as well
	<-timer.C

	runtime := time.Since(start).Seconds()

	if runtime < 0.009 {
		t.Errorf("took ~%v milliseconds, should be ~9 milliseconds\n", runtime*1000)
	}
}

func TestTimer_Stop(t *testing.T) {
	timer := synchronization.NewTimer(2 * time.Millisecond)
	time.Sleep(3 * time.Millisecond)
	timer.Stop()
	releaseC := make(chan bool, 1)
	time.AfterFunc(1*time.Millisecond, func() { releaseC <- true })

	select {
	case <-timer.C:
		t.Error("timer pinged, but it was stopped already")
	case <-releaseC:
	}
}

func TestTimer_GetTimer(t *testing.T) {
	x := synchronization.NewTimer(time.Second)
	require.NotNil(t, x, "cannot create new timer")

	timer := x.GetTimer()
	require.NotNil(t, timer, "timer inner object was not created, something is really wrong")
}

func TestValidTimerStopOp(t *testing.T) {
	// testing that a timer stopped before it was expired does not later ping
	timer := synchronization.NewTimer(1 * time.Millisecond)
	timer.Stop()
	releaseC := make(chan bool, 1)
	time.AfterFunc(2*time.Millisecond, func() { releaseC <- true })

	select {
	case <-timer.C:
		t.Error("timer pinged, but it was stopped already")
	case <-releaseC:
	}
}

func TestValidTimerStopOpDoesNotBlock(t *testing.T) {
	// testing that a timer stopped after it pings, is not blocking
	timer := synchronization.NewTimer(1 * time.Millisecond)
	<-timer.C
	timer.Stop()
}

func TestValidTimerResetOp(t *testing.T) {
	// testing that a timer reset before it was expired does not ping too early
	start := time.Now()

	timer := synchronization.NewTimer(2 * time.Millisecond)
	timer.Reset(3 * time.Millisecond)
	<-timer.C

	runtime := time.Since(start).Seconds()

	if runtime < 0.003 {
		t.Errorf("took ~%v milliseconds, should be ~3 milliseconds\n", runtime*1000)
	}
}

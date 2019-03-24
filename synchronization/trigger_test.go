// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package synchronization_test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type report struct {
	message string
	fields  []*log.Field
}

type collector struct {
	errors chan report
}

func (c *collector) Error(message string, fields ...*log.Field) {
	c.errors <- report{message, fields}
}

func mockLogger() *collector {
	c := &collector{errors: make(chan report)}
	return c
}

func TestPeriodicalTriggerStartsOk(t *testing.T) {
	logger := mockLogger()
	fromTrigger := make(chan struct{})
	stop := make(chan struct{})
	trigger := func() {
		select {
		case fromTrigger <- struct{}{}:
		case <-stop:
			return
		}
	}
	tickTime := time.Microsecond
	p := synchronization.NewPeriodicalTrigger(context.Background(), tickTime, logger, trigger, nil)

	<-fromTrigger // test will block if the trigger did not happen

	close(stop)
	p.Stop()
}

func TestTriggerInternalMetrics(t *testing.T) {
	logger := mockLogger()
	fromTrigger := make(chan struct{})
	stop := make(chan struct{})
	trigger := func() {
		select {
		case fromTrigger <- struct{}{}:
		case <-stop:
			return
		}
	}
	tickTime := time.Microsecond
	p := synchronization.NewPeriodicalTrigger(context.Background(), tickTime, logger, trigger, nil)

	// wait for three triggers
	for i := 0; i < 3; i++ {
		<-fromTrigger
	}

	time.Sleep(time.Millisecond) // yield
	require.EqualValues(t, 3, p.TimesTriggered(), "expected 3 triggers but got %d (metric)", p.TimesTriggered())
	close(stop)
	p.Stop()
}

func TestPeriodicalTrigger_Stop(t *testing.T) {
	logger := mockLogger()
	x := 0
	p := synchronization.NewPeriodicalTrigger(context.Background(), time.Millisecond*2, logger, func() { x++ }, nil)
	p.Stop()
	time.Sleep(3 * time.Millisecond)
	require.Equal(t, 0, x, "expected no ticks")
}

func TestPeriodicalTrigger_StopAfterTrigger(t *testing.T) {
	logger := mockLogger()
	x := 0
	p := synchronization.NewPeriodicalTrigger(context.Background(), time.Millisecond, logger, func() { x++ }, nil)
	time.Sleep(time.Microsecond * 1100)
	p.Stop()
	xValueOnStop := x
	time.Sleep(time.Millisecond * 5)
	require.Equal(t, xValueOnStop, x, "expected one tick due to stop")
}

func TestPeriodicalTriggerStopOnContextCancel(t *testing.T) {
	logger := mockLogger()
	ctx, cancel := context.WithCancel(context.Background())
	x := 0
	synchronization.NewPeriodicalTrigger(ctx, time.Millisecond*2, logger, func() { x++ }, nil)
	cancel()
	time.Sleep(3 * time.Millisecond)
	require.Equal(t, 0, x, "expected no ticks")
}

func TestPeriodicalTriggerStopWorksWhenContextIsCancelled(t *testing.T) {
	logger := mockLogger()
	ctx, cancel := context.WithCancel(context.Background())
	x := 0
	p := synchronization.NewPeriodicalTrigger(ctx, time.Millisecond*2, logger, func() { x++ }, nil)
	cancel()
	time.Sleep(3 * time.Millisecond)
	require.Equal(t, 0, x, "expected no ticks")
	p.Stop()
	require.Equal(t, 0, x, "expected stop to not block")
}

func TestPeriodicalTriggerStopOnContextCancelWithStopAction(t *testing.T) {
	logger := mockLogger()
	ctx, cancel := context.WithCancel(context.Background())
	x := 0
	synchronization.NewPeriodicalTrigger(ctx, time.Millisecond*2, logger, func() { x++ }, func() { x = 20 })
	cancel()
	time.Sleep(time.Millisecond) // yield
	require.Equal(t, 20, x, "expected x to have the stop value")
}

func TestPeriodicalTriggerRunsOnStopAction(t *testing.T) {
	logger := mockLogger()
	latch := make(chan struct{})
	x := 0
	p := synchronization.NewPeriodicalTrigger(context.Background(),
		time.Second,
		logger,
		func() { x++ },
		func() {
			x = 20
			latch <- struct{}{}
		})
	p.Stop()
	<-latch // wait for stop to happen...
	require.Equal(t, 20, x, "expected x to have the stop value")
}

func TestPeriodicalTriggerKeepsGoingOnPanic(t *testing.T) {
	logger := mockLogger()
	x := 0
	p := synchronization.NewPeriodicalTrigger(context.Background(),
		time.Millisecond,
		logger,
		func() {
			x++
			panic("we should not see this other than the logs")
		},
		nil)

	// more than one error means more than one panic means it recovers correctly
	for i := 0; i < 2; i++ {
		<-logger.errors
	}

	p.Stop()

	require.True(t, x > 1, "expected trigger to have ticked more than once (even though it panics) but it ticked %d", x)
}

package synchronization_test

import (
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/stretchr/testify/require"
	"sync/atomic"
	"testing"
	"time"
)

func getExpected(startTime, endTime time.Time, tickTime time.Duration) uint32 {
	duration := endTime.Sub(startTime)
	expected := uint32((duration.Seconds() * 1000) / (tickTime.Seconds() * 1000))
	return expected
}

func TestNewTrigger(t *testing.T) {
	p := synchronization.NewTrigger(time.Duration(5), func() {})
	require.NotNil(t, p, "failed to initialize the ticker")
	require.False(t, p.IsRunning(), "should not be running when created")
}

func TestNewPeriodicalTrigger(t *testing.T) {
	p := synchronization.NewPeriodicalTrigger(time.Duration(5), func() {})
	require.NotNil(t, p, "failed to initialize the ticker")
	require.False(t, p.IsRunning(), "should not be running when created")
}

func TestTrigger_FiresOnlyOnce(t *testing.T) {
	x := 0
	p := synchronization.NewTrigger(time.Millisecond*1, func() { x++ })
	p.Start()
	time.Sleep(time.Millisecond * 10)
	require.Equal(t, 1, x, "expected one tick")
}

func TestPeriodicalTrigger_NoStartDoesNotFireFunc(t *testing.T) {
	x := 0
	p := synchronization.NewPeriodicalTrigger(time.Millisecond*1, func() { x++ })
	time.Sleep(time.Millisecond * 10)
	require.Equal(t, 0, x, "expected no ticks")
	p.Stop() // to hold the reference
}

func TestPeriodicalTrigger_Start(t *testing.T) {
	var x uint32
	start := time.Now()
	tickTime := 5 * time.Millisecond
	p := synchronization.NewPeriodicalTrigger(tickTime, func() { atomic.AddUint32(&x, 1) })
	p.Start()
	time.Sleep(time.Millisecond * 30)
	expected := getExpected(start, time.Now(), tickTime)
	require.True(t, expected/2 < atomic.LoadUint32(&x), "expected more than %d ticks, but got %d", expected/2, atomic.LoadUint32(&x))
	p.Stop()
}

func TestTriggerInternalMetrics(t *testing.T) {
	var x uint32
	start := time.Now()
	tickTime := 5 * time.Millisecond
	p := synchronization.NewPeriodicalTrigger(tickTime, func() { atomic.AddUint32(&x, 1) })
	p.Start()
	time.Sleep(time.Millisecond * 30)
	expected := getExpected(start, time.Now(), tickTime)
	require.True(t, expected/2 < atomic.LoadUint32(&x), "expected more than %d ticks, but got %d", expected/2, atomic.LoadUint32(&x))
	require.True(t, uint64(expected/2) < p.TimesTriggered(), "expected more than %d ticks, but got %d (metric)", expected/2, p.TimesTriggered())
	p.Stop()
}

func TestPeriodicalTrigger_Reset(t *testing.T) {
	x := 0
	p := synchronization.NewPeriodicalTrigger(time.Millisecond*2, func() { x++ })
	p.Start()
	p.Reset(time.Millisecond * 3)
	require.Equal(t, 0, x, "expected zero ticks for now")
	time.Sleep(time.Millisecond * 5)
	require.Equal(t, 1, x, "expected one ticks with new reset value")
}

func TestPeriodicalTrigger_FireNow(t *testing.T) {
	x := 0
	p := synchronization.NewPeriodicalTrigger(time.Millisecond*50, func() { x++ })
	p.Start()
	time.Sleep(time.Millisecond * 25)
	p.FireNow()
	time.Sleep(time.Millisecond * 35)
	// at this point we are 50+ ms into the logic, we should see only one tick as we reset though firenow midway
	require.Equal(t, 1, x, "expected one tick for now")
	require.EqualValues(t, 0, p.TimesTriggered(), "expected to not have a timer tick trigger now, got %d ticks", p.TimesTriggered())
	require.EqualValues(t, 0, p.TimesReset(), "should not count a reset on firenow")
	require.EqualValues(t, 1, p.TimesTriggeredManually(), "we triggered manually once")
}

func TestPeriodicalTrigger_Stop(t *testing.T) {
	x := 0
	p := synchronization.NewPeriodicalTrigger(time.Millisecond*2, func() { x++ })
	p.Start()
	p.Stop()
	require.Equal(t, 0, x, "expected no ticks")
}

func TestPeriodicalTrigger_StopAfterTrigger(t *testing.T) {
	x := 0
	p := synchronization.NewPeriodicalTrigger(time.Millisecond, func() { x++ })
	p.Start()
	time.Sleep(time.Microsecond * 1100)
	p.Stop()
	time.Sleep(time.Millisecond * 2)
	require.Equal(t, 1, x, "expected one tick due to stop")
}

package synchronization_test

import (
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

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
	time.Sleep(time.Millisecond * 3)
	require.Equal(t, 1, x, "expected no ticks")
}

func TestPeriodicalTrigger_NoStartDoesNotFireFunc(t *testing.T) {
	x := 0
	p := synchronization.NewPeriodicalTrigger(time.Millisecond*1, func() { x++ })
	time.Sleep(time.Millisecond * 3)
	require.Equal(t, 0, x, "expected no ticks")
	p.Stop() // to hold the reference
}

func TestPeriodicalTrigger_Start(t *testing.T) {
	x := 0
	p := synchronization.NewPeriodicalTrigger(time.Millisecond*4, func() { x++ })
	p.Start()
	time.Sleep(time.Millisecond * 10)
	require.Equal(t, 2, x, "expected two ticks")
}

func TestTriggerInternalMetrics(t *testing.T) {
	x := 0
	p := synchronization.NewPeriodicalTrigger(time.Millisecond*4, func() { x++ })
	p.Start()
	time.Sleep(time.Millisecond * 10)
	require.Equal(t, 2, x, "expected two ticks")
	require.EqualValues(t, 2, p.TimesTriggered(), "expected two ticks")
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
	p := synchronization.NewPeriodicalTrigger(time.Millisecond*2, func() { x++ })
	p.Start()
	p.FireNow()
	time.Sleep(time.Millisecond)
	require.Equal(t, 1, x, "expected one ticks for now")
	time.Sleep(time.Microsecond * 1500)
	// at this point ~2.5 millis should have passed after the internal reset that happend on firenow
	require.Equal(t, 2, x, "expected two ticks")
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
	time.Sleep(time.Microsecond * 1500)
	p.Stop()
	time.Sleep(time.Millisecond * 2)
	require.Equal(t, 1, x, "expected one ticks due to stop")
}

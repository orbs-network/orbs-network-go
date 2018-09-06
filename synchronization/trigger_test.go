package synchronization_test

import (
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNewPeriodicalTimer(t *testing.T) {
	p := synchronization.NewPeriodicalTimer(time.Duration(5), func() {})
	require.NotNil(t, p, "failed to initialize the ticker")
	require.False(t, p.IsRunning(), "should not be running when created")
}

func TestPeriodicalTrigger_Start(t *testing.T) {
	x := 0
	p := synchronization.NewPeriodicalTimer(time.Millisecond*2, func() { x++ })
	p.Start()
	time.Sleep(time.Millisecond * 5)
	require.Equal(t, 2, x, "expected two ticks")
}

func TestTriggerInternalMetrics(t *testing.T) {
	x := 0
	p := synchronization.NewPeriodicalTimer(time.Millisecond*2, func() { x++ })
	p.Start()
	time.Sleep(time.Millisecond * 5)
	require.Equal(t, 2, x, "expected two ticks")
	require.EqualValues(t, 2, p.TimesTriggered(), "expected two ticks")
}

func TestPeriodicalTrigger_Reset(t *testing.T) {
	x := 0
	p := synchronization.NewPeriodicalTimer(time.Millisecond*2, func() { x++ })
	p.Start()
	p.Reset(time.Millisecond * 3)
	require.Equal(t, 0, x, "expected zero ticks for now")
	time.Sleep(time.Millisecond * 5)
	require.Equal(t, 1, x, "expected one ticks with new reset value")
}

func TestPeriodicalTrigger_FireNow(t *testing.T) {
	x := 0
	p := synchronization.NewPeriodicalTimer(time.Millisecond*2, func() { x++ })
	p.Start()
	p.FireNow()
	time.Sleep(time.Millisecond)
	require.Equal(t, 1, x, "expected one ticks for now")
	time.Sleep(time.Microsecond * 1500)
	// at this point ~2.5 millis should have passed after the internal reset that happend on firenow
	require.Equal(t, 2, x, "expected two ticks")
	require.EqualValues(t, 1, p.TimesReset(), "should reset internally on immediate trigger")
	require.EqualValues(t, 1, p.TimesTriggeredManually(), "we triggered manually once")
}

func TestPeriodicalTrigger_Stop(t *testing.T) {
	x := 0
	p := synchronization.NewPeriodicalTimer(time.Millisecond*2, func() { x++ })
	p.Start()
	p.Stop()
	require.Equal(t, 0, x, "expected no ticks")
}

func TestPeriodicalTrigger_StopAfterTrigger(t *testing.T) {
	x := 0
	p := synchronization.NewPeriodicalTimer(time.Millisecond, func() { x++ })
	p.Start()
	time.Sleep(time.Microsecond * 1500)
	p.Stop()
	time.Sleep(time.Millisecond * 2)
	require.Equal(t, 1, x, "expected one ticks due to stop")
}

package metric

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestRate_Measure(t *testing.T) {
	t.Skip("this test is for Shai")

	rate := newRateWithInterval("tps", 10*time.Millisecond)
	baseRate := rate.movingAverage.Value()

	require.EqualValues(t, 0, baseRate)
	for i := 0; i < 100; i++ {
		rate.Measure(1)
	}

	time.Sleep(10 * time.Millisecond)
	require.EqualValues(t, 100, baseRate)

	time.Sleep(10 * time.Millisecond)
	require.EqualValues(t, 50, baseRate)
}

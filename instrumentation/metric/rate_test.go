package metric

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestRate_Measure(t *testing.T) {
	start := time.Now()
	rate := newRateWihStart("tps", start)

	require.EqualValues(t, 0, rate.export().Rate)
	rate.Measure(100)

	rate.maybeRotateAsOf(start.Add(1100 * time.Millisecond))

	require.EqualValues(t, 100, rate.export().Rate)

	for i := 1; i < 10; i++ {
		rate.maybeRotateAsOf(start.Add(time.Duration(i) * time.Second))
	}

	require.Condition(t, func() bool {
		return rate.export().Rate < 100
	}, "rate did not decay")

}

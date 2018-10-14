package metric

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGauge_Add(t *testing.T) {
	g := Gauge{}
	g.AddUint32(10)

	require.EqualValues(t, 10, g.Value(), "gauge value differed from expected")
}

func TestGauge_Inc(t *testing.T) {
	g := Gauge{}
	g.Inc()

	require.EqualValues(t, 1, g.Value(), "gauge value differed from expected")
}

func TestGauge_Dec(t *testing.T) {
	g := Gauge{}
	g.Inc()
	g.Dec()

	require.EqualValues(t, 0, g.Value(), "gauge value differed from expected")
}

func TestGauge_SubUint32(t *testing.T) {
	g := Gauge{}
	g.AddUint32(10)
	g.SubUint32(10)

	require.EqualValues(t, 0, g.Value(), "gauge value differed from expected")
}


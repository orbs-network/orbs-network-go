package metric

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInMemoryRegistry_ExportAll(t *testing.T) {
	registry := NewRegistry()
	gauge := registry.NewGauge("hello")
	gauge.Add(1)

	gaugeValue := registry.ExportAll()["hello"].(gaugeExport)
	require.EqualValues(t, gaugeValue.Value, 1)
}

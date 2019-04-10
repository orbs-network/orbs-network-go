package metric

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

/**
Format reference: https://prometheus.io/docs/instrumenting/exposition_formats/
*/
func Test_PrometheusFormatterForGauge(t *testing.T) {
	r := NewRegistry()
	status := r.NewGauge("Ethereum.Node.LastBlock")

	result := r.ExportPrometheus()

	require.Regexp(t, "# TYPE Ethereum_Node_LastBlock gauge", result)
	require.Regexp(t, "Ethereum_Node_LastBlock 0", result)

	status.Update(5123441)
	updatedResult := r.ExportPrometheus()
	require.Regexp(t, "Ethereum_Node_LastBlock 5123441", updatedResult)
}

func Test_PrometheusFormatterForGaugeWithParams(t *testing.T) {
	bytes, _ := hex.DecodeString("0123456789abcdef")
	r := NewRegistry().WithVirtualChainId(100000).WithNodeAddress(primitives.NodeAddress(bytes))
	status := r.NewGauge("Ethereum.Node.LastBlock")
	status.Update(5123441)

	resultWithParams := r.ExportPrometheus()
	require.Regexp(t, "Ethereum_Node_LastBlock{vcid=\"100000\",node=\"0123456789abcdef\"} 5123441", resultWithParams)
}

func Test_PrometheusFormatterForHistogram(t *testing.T) {
	r := NewRegistry()
	status := r.NewHistogram("Some.Latency", 10000)

	result := r.ExportPrometheus()

	require.Regexp(t, "# TYPE Some_Latency histogram", result)
	require.Regexp(t, "Some_Latency{quantile=\"0.01\"} 0", result)
	require.Regexp(t, "Some_Latency{quantile=\"0.5\"} 0", result)
	require.Regexp(t, "Some_Latency{quantile=\"0.99\"} 0", result)
	require.Regexp(t, "Some_Latency{quantile=\"0.999\"} 0$", result)

	status.RecordSince(time.Now())
	updatedResult := r.ExportPrometheus()
	t.Log(result)
	require.Regexp(t, "Some_Latency{quantile=\"0.01\"} 0.00", updatedResult)
	require.Regexp(t, "Some_Latency{quantile=\"0.5\"} 0.00", updatedResult)
	require.Regexp(t, "Some_Latency{quantile=\"0.99\"} 0.00", updatedResult)
	require.Regexp(t, "Some_Latency{quantile=\"0.999\"} 0.00", updatedResult)
}

func Test_PrometheusFormatterForHistogramWithParams(t *testing.T) {
	r := NewRegistry().WithVirtualChainId(100000)
	status := r.NewHistogram("Some.Latency", 10000)
	status.RecordSince(time.Now())

	resultWithParams := r.ExportPrometheus()
	require.Regexp(t, "Some_Latency{vcid=\"100000\",quantile=\"0.01\"} 0.00", resultWithParams)
}

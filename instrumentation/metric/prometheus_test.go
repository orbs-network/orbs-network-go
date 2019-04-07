package metric

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func Test_PrometheusFormatterForGauge(t *testing.T) {
	r := NewRegistry()
	status := r.NewGauge("Ethereum.Node.LastBlock")

	result := r.ExportPrometheus()

	require.Regexp(t, "Ethereum_Node_LastBlock 0", result)

	status.Update(5123441)
	updatedResult := r.ExportPrometheus()
	require.Regexp(t, "Ethereum_Node_LastBlock 5123441", updatedResult)
}

func Test_PrometheusFormatterForHistogram(t *testing.T) {
	r := NewRegistry()
	status := r.NewHistogram("Some.Latency", 10000)

	result := r.ExportPrometheus()

	require.Regexp(t, "Some_Latency{quantile=\"0.01\"} 0", result)
	require.Regexp(t, "Some_Latency{quantile=\"0.5\"} 0", result)
	require.Regexp(t, "Some_Latency{quantile=\"0.99\"} 0", result)
	require.Regexp(t, "Some_Latency{quantile=\"0.999\"} 0$", result)

	status.RecordSince(time.Now())
	updatedResult := r.ExportPrometheus()

	require.Regexp(t, "Some_Latency{quantile=\"0.01\"} 0.000", updatedResult)
	require.Regexp(t, "Some_Latency{quantile=\"0.5\"} 0.000", updatedResult)
	require.Regexp(t, "Some_Latency{quantile=\"0.99\"} 0.000", updatedResult)
	require.Regexp(t, "Some_Latency{quantile=\"0.999\"} 0.000", updatedResult)

}

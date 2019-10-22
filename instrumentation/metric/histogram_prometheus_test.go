//+build !race
// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package metric

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// This does NOT test correctness of Histogram
// (e.g. that calculation of quantiles for given values is correct)
// It only verifies the accurate conversion of metric values into Prometheus format.
func Test_PrometheusFormatterForHistogram(t *testing.T) {
	r := NewRegistry()
	const SEC = int64(time.Second)
	histo := r.NewHistogram("Some.Latency", 1000*SEC)

	for i := 0; i < 1000; i++ {
		histo.Record(int64(i) * SEC)
	}

	metrics := r.ExportAll()
	prometheusStrings := MetricsToPrometheusStrings(metrics, nil)
	promStr := r.ExportPrometheus()
	fmt.Println(prometheusStrings)
	require.Regexp(t, "# TYPE Some_Latency histogram", promStr)
	for _, row := range metrics["Some.Latency"].PrometheusRow() {
		expectedStr := fmt.Sprintf("Some_Latency{quantile=\"%s\"} %s",
			QuantileAsStr(row.quantile), row.value)
		require.Regexp(t, expectedStr, promStr)
	}
}

func Test_PrometheusFormatterForHistogramWithParams(t *testing.T) {
	r := NewRegistry().WithVirtualChainId(100000)
	const SEC = int64(time.Second)
	histo := r.NewHistogram("Some.Latency", 1000*SEC)

	for i := 0; i < 1000; i++ {
		histo.Record(int64(i) * SEC)
	}

	metrics := r.ExportAll()
	prometheusStrings := MetricsToPrometheusStrings(metrics, nil)
	promStr := r.ExportPrometheus()
	fmt.Println(prometheusStrings)
	require.Regexp(t, "# TYPE Some_Latency histogram", promStr)
	for _, row := range metrics["Some.Latency"].PrometheusRow() {
		expectedStr := fmt.Sprintf("Some_Latency{vcid=\"100000\",quantile=\"%s\"} %s",
			QuantileAsStr(row.quantile), row.value)
		require.Regexp(t, expectedStr, promStr)
	}

	//status := r.NewHistogram("Some.Latency", 10000)
	//status.RecordSince(time.Now())
	//
	//resultWithParams := r.ExportPrometheus()
	//require.Regexp(t, "Some_Latency{vcid=\"100000\",quantile=\"0.01\"} 0.00", resultWithParams)
}

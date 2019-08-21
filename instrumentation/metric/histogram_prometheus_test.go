//+build !race
// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package metric

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

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

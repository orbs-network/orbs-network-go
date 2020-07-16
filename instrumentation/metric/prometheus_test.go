//+build !race
// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package metric

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

/**
Format reference: https://prometheus.io/docs/instrumenting/exposition_formats/
*/

// This does NOT test correctness of Histogram
// (e.g. that calculation of quantiles for given values is correct)
// It only verifies the accurate conversion of metric values into Prometheus format.
func TestHistogram_ExportPrometheusFormatter(t *testing.T) {
	r := NewRegistry().WithVirtualChainId(100000)
	const SEC = int64(time.Second)
	histo := r.NewHistogram("Some.Latency", 1000*SEC)

	for i := 0; i < 1000; i++ {
		histo.Record(int64(i) * SEC)
	}

	promStr := r.ExportPrometheus()

	require.Regexp(t, "# TYPE Some_Latency histogram", promStr)
	require.Equal(t, 7, strings.Count(promStr, "Some_Latency{vcid=\"100000\",aggregation="))
	// ugly but simple
	require.Equal(t, `# TYPE Some_Latency histogram
Some_Latency{vcid="100000",aggregation="min"} 0
Some_Latency{vcid="100000",aggregation="median"} 515396.075519
Some_Latency{vcid="100000",aggregation="95p"} 962072.674303
Some_Latency{vcid="100000",aggregation="99p"} 996432.412671
Some_Latency{vcid="100000",aggregation="max"} 1030792.151039
Some_Latency{vcid="100000",aggregation="avg"} 499547.39453952
Some_Latency{vcid="100000",aggregation="count"} 1000
`, promStr)
}

func TestGauge_ExportPrometheus(t *testing.T) {
	r := NewRegistry()
	status := r.NewGauge("Ethereum.Node.LastBlock")

	result := r.ExportPrometheus()

	require.Regexp(t, "# TYPE Ethereum_Node_LastBlock gauge", result)
	require.Regexp(t, "Ethereum_Node_LastBlock 0", result)

	status.Update(5123441)
	updatedResult := r.ExportPrometheus()
	require.Regexp(t, "Ethereum_Node_LastBlock 5123441", updatedResult)
}

func TestGauge_ExportPrometheusWithLabels(t *testing.T) {
	bytes, _ := hex.DecodeString("0123456789abcdef")
	r := NewRegistry().WithVirtualChainId(100000).WithNodeAddress(primitives.NodeAddress(bytes))
	status := r.NewGauge("Ethereum.Node.LastBlock")
	status.Update(5123441)

	resultWithLabels := r.ExportPrometheus()
	require.Regexp(t, "Ethereum_Node_LastBlock{vcid=\"100000\",node=\"0123456789abcdef\"} 5123441", resultWithLabels)
}

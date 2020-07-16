//+build !race
// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package metric

import (
	"fmt"
	"github.com/codahale/hdrhistogram"
	"strconv"
	"sync/atomic"
	"time"
)

type Histogram struct {
	name 		  string
	histo         *hdrhistogram.WindowedHistogram
	overflowCount int64
}

type histogramExport struct {
	Name    string
	Min     float64
	P50     float64
	P95     float64
	P99     float64
	Max     float64
	Avg     float64
	Samples int64
}

func newHistogram(name string, max int64, n int) *Histogram {
	return &Histogram{
		name:  name,
		histo: hdrhistogram.NewWindowed(n, 0, max, 1),
	}
}

func (h *Histogram) Name() string {
	return h.name
}

func (h *Histogram) RecordSince(t time.Time) {
	d := time.Since(t).Nanoseconds()
	if err := h.histo.Current.RecordValue(int64(d)); err != nil {
		atomic.AddInt64(&h.overflowCount, 1)
	}
}

func (h *Histogram) Record(measurement int64) {
	if err := h.histo.Current.RecordValue(measurement); err != nil {
		atomic.AddInt64(&h.overflowCount, 1)
	}
}

func (h *Histogram) CurrentSamples() int64 {
	histo := h.histo.Current
	return histo.TotalCount()
}

func (h *Histogram) Value() interface{} {
	return nil
}

func (h *Histogram) Export() exportedMetric {
	histo := h.histo.Merge()

	return &histogramExport{
		h.name,
		toMillis(histo.Min()),
		toMillis(histo.ValueAtQuantile(50)),
		toMillis(histo.ValueAtQuantile(95)),
		toMillis(histo.ValueAtQuantile(99)),
		toMillis(histo.Max()),
		floatToMillis(histo.Mean()),
		histo.TotalCount(),
	}
}

func (h *Histogram) Rotate() {
	h.histo.Rotate()
}

// Note: in real life we have labels
// this is here because there is a different implementation for races
func (h *Histogram) exportPrometheus(labelString string) string {
	histo := h.histo.Merge()
	typeRow := prometheusType(prometheusName(h.name), "histogram")
	valueMinRow := fmt.Sprintf("%s{%s,aggregation=\"min\"} %s\n", prometheusName(h.name), labelString, strconv.FormatFloat(toMillis(histo.Min()), 'f', -1, 64))
	valueMeanRow := fmt.Sprintf("%s{%s,aggregation=\"median\"} %s\n", prometheusName(h.name), labelString, strconv.FormatFloat(toMillis(histo.ValueAtQuantile(50)), 'f', -1, 64))
	value95Row := fmt.Sprintf("%s{%s,aggregation=\"95p\"} %s\n", prometheusName(h.name), labelString, strconv.FormatFloat(toMillis(histo.ValueAtQuantile(95)), 'f', -1, 64))
	value99Row := fmt.Sprintf("%s{%s,aggregation=\"99p\"} %s\n", prometheusName(h.name), labelString, strconv.FormatFloat(toMillis(histo.ValueAtQuantile(99)), 'f', -1, 64))
	valueMaxRow := fmt.Sprintf("%s{%s,aggregation=\"max\"} %s\n", prometheusName(h.name), labelString, strconv.FormatFloat(toMillis(histo.Max()), 'f', -1, 64))
	valueAvgRow := fmt.Sprintf("%s{%s,aggregation=\"avg\"} %s\n", prometheusName(h.name), labelString, strconv.FormatFloat(floatToMillis(histo.Mean()), 'f', -1, 64))
	valueCountRow := fmt.Sprintf("%s{%s,aggregation=\"count\"} %s\n", prometheusName(h.name), labelString, strconv.FormatInt(histo.TotalCount(), 10))
	return typeRow + valueMinRow + valueMeanRow + value95Row + value99Row + valueMaxRow + valueAvgRow + valueCountRow
}

func toMillis(nanoseconds int64) float64 {
	return floatToMillis(float64(nanoseconds))
}

func floatToMillis(nanoseconds float64) float64 {
	return nanoseconds / 1e+6
}

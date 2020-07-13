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
	"sync/atomic"
	"time"
)

type HistogramTimeDiff struct {
	namedMetric
	histo         *hdrhistogram.WindowedHistogram
	overflowCount int64
}

func newHistogramTimeDiff(name string, max int64, n int) *HistogramTimeDiff {
	return &HistogramTimeDiff{
		namedMetric: namedMetric{name: name},
		histo:       hdrhistogram.NewWindowed(n, 0, max, 1),
	}
}

func (h *HistogramTimeDiff) RecordSince(t time.Time) {
	d := time.Since(t).Nanoseconds()
	if err := h.histo.Current.RecordValue(int64(d)); err != nil {
		atomic.AddInt64(&h.overflowCount, 1)
	}
}

func (h *HistogramTimeDiff) String() string {
	var errorRate float64
	histo := h.histo.Current

	if h.overflowCount > 0 {
		errorRate = float64(histo.TotalCount()) / float64(h.overflowCount)
	} else {
		errorRate = 0
	}

	return fmt.Sprintf(
		"metric %s: [min=%f, p50=%f, p95=%f, p99=%f, max=%f, avg=%f, samples=%d, error rate=%f]\n",
		h.name,
		toMillis(histo.Min()),
		toMillis(histo.ValueAtQuantile(50)),
		toMillis(histo.ValueAtQuantile(95)),
		toMillis(histo.ValueAtQuantile(99)),
		toMillis(histo.Max()),
		floatToMillis(histo.Mean()),
		histo.TotalCount(),
		errorRate)
}

func (h *HistogramTimeDiff) Export() exportedMetric {
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

func (h *HistogramTimeDiff) Rotate() {
	h.histo.Rotate()
}

type HistogramInt64 struct { // TODO DRY HistogramTimeDiff is similar
	namedMetric
	histo         *hdrhistogram.WindowedHistogram
	overflowCount int64
}

func newHistogramInt64(name string, max int64, n int) *HistogramInt64 {
	return &HistogramInt64{
		namedMetric: namedMetric{name: name},
		histo:       hdrhistogram.NewWindowed(n, 0, max, 1),
	}
}

func (h *HistogramInt64) Record(measurement int64) {
	if err := h.histo.Current.RecordValue(measurement); err != nil {
		atomic.AddInt64(&h.overflowCount, 1)
	}
}

func (h *HistogramInt64) String() string {
	var errorRate float64
	histo := h.histo.Current

	if h.overflowCount > 0 {
		errorRate = float64(histo.TotalCount()) / float64(h.overflowCount)
	} else {
		errorRate = 0
	}

	return fmt.Sprintf(
		"metric %s: [min=%d, p50=%d, p95=%d, p99=%d, max=%d, avg=%f, samples=%d, error rate=%f]\n",
		h.name,
		histo.Min(),
		histo.ValueAtQuantile(50),
		histo.ValueAtQuantile(95),
		histo.ValueAtQuantile(99),
		histo.Max(),
		histo.Mean(),
		histo.TotalCount(),
		errorRate)
}

func (h *HistogramInt64) Export() exportedMetric {
	histo := h.histo.Merge()

	return &histogramExport{
		h.name,
		float64(histo.Min()),
		float64(histo.ValueAtQuantile(50)),
		float64(histo.ValueAtQuantile(95)),
		float64(histo.ValueAtQuantile(99)),
		float64(histo.Max()),
		histo.Mean(),
		histo.TotalCount(),
	}
}

func (h *HistogramInt64) Rotate() {
	h.histo.Rotate()
}

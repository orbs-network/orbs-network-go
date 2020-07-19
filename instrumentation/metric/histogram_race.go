//+build race
// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package metric

import (
	"time"
)

type Histogram struct {
	name 		  string
}

type histogramExport struct {
}

func newHistogram(name string, max int64, n int) *Histogram {
	return &Histogram{name: name}
}

func (h *Histogram) Name() string {
	return h.name
}

func (h Histogram) Export() exportedMetric {
	return &histogramExport{}
}

func (h *Histogram) CurrentSamples() int64 {
	return 0
}

func (h *Histogram) Value() interface{} {
	return nil
}

func (h *Histogram) Rotate() {
}

func (h *Histogram) RecordSince(t time.Time) {
}

func (h *Histogram) Record(measurement int64) {
}

// Note: in real life we have labels
func (h *Histogram) exportPrometheus(labelString string) string {
	return ""
}

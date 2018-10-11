package metric

import (
	"fmt"
	"github.com/codahale/hdrhistogram"
	"sync/atomic"
	"time"
)

type Histogram struct {
	name          string
	histo         *hdrhistogram.Histogram
	overflowCount int64
}

func newHistogram(name string, max int64) *Histogram {
	return &Histogram{
		name:  name,
		histo: hdrhistogram.New(0, max, 1),
	}
}

func (h *Histogram) RecordMillisSince(t time.Time) {
	d := time.Since(t)
	if err := h.histo.RecordValue(int64(d) / 1000 / 1000); err != nil {
		atomic.AddInt64(&h.overflowCount, 1)
	}
}

func (h *Histogram) String() string {
	var errorRate float64
	if h.overflowCount > 0 {
		errorRate = float64(h.histo.TotalCount())/float64(h.overflowCount)
	} else {
		errorRate = 0
	}

	return fmt.Sprintf(
		"metric %s: [min=%d, p50=%d, p95=%d, p99=%d, max=%d, avg=%f, samples=%d, error rate=%f]\n",
		h.name,
		h.histo.Min(),
		h.histo.ValueAtQuantile(50),
		h.histo.ValueAtQuantile(95),
		h.histo.ValueAtQuantile(99),
		h.histo.Max(),
		h.histo.Mean(),
		h.histo.TotalCount(),
		errorRate)
}


package metric

import (
	"fmt"
	"github.com/codahale/hdrhistogram"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"sync/atomic"
	"time"
)

type Histogram struct {
	namedMetric
	histo         *hdrhistogram.Histogram
	overflowCount int64
}

type histogramExport struct {
	Name    string
	Min     int64
	P50     int64
	P95     int64
	P99     int64
	Max     int64
	Avg     float64
	Samples int64
}

func newHistogram(name string, max int64) *Histogram {
	return &Histogram{
		namedMetric: namedMetric{name: name},
		histo:       hdrhistogram.New(0, max, 1),
	}
}

func (h *Histogram) RecordSince(t time.Time) {
	d := time.Since(t)
	if err := h.histo.RecordValue(int64(d)); err != nil {
		atomic.AddInt64(&h.overflowCount, 1)
	}
}

func (h *Histogram) String() string {
	var errorRate float64
	if h.overflowCount > 0 {
		errorRate = float64(h.histo.TotalCount()) / float64(h.overflowCount)
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

func (h *Histogram) Export() exportedMetric {
	return histogramExport{
		h.name,
		h.histo.Min(),
		h.histo.ValueAtQuantile(50),
		h.histo.ValueAtQuantile(95),
		h.histo.ValueAtQuantile(99),
		h.histo.Max(),
		h.histo.Mean(),
		h.histo.TotalCount(),
	}
}

func (h *Histogram) Reset() {
	h.histo.Reset()
}

func (h histogramExport) LogRow() []*log.Field {
	return []*log.Field{
		log.String("metric", h.Name),
		log.String("metric-type", "histogram"),
		log.Int64("min", h.Min),
		log.Int64("p50", h.P50),
		log.Int64("p95", h.P95),
		log.Int64("p99", h.P99),
		log.Int64("max", h.Max),
		log.Float64("avg", h.Avg),
		log.Int64("samples", h.Samples),
	}
}

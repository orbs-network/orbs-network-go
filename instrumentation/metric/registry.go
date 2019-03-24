// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package metric

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"sync"
	"time"
)

const REPORT_INTERVAL = 30 * time.Second
const AGGREGATION_SPAN = 10 * time.Minute

type Factory interface {
	NewHistogram(name string, maxValue int64) *Histogram
	NewLatency(name string, maxDuration time.Duration) *Histogram
	NewGauge(name string) *Gauge
	NewRate(name string) *Rate
	NewText(name string, defaultValue ...string) *Text
}

type Registry interface {
	Factory
	String() string
	ExportAll() map[string]exportedMetric
	PeriodicallyReport(ctx context.Context, logger log.BasicLogger)
}

type exportedMetric interface {
	LogRow() []*log.Field
}

type metric interface {
	fmt.Stringer
	Name() string
	Export() exportedMetric
}

type namedMetric struct {
	name string
}

func (m *namedMetric) Name() string {
	return m.name
}

func NewRegistry() Registry {
	return &inMemoryRegistry{}
}

type inMemoryRegistry struct {
	mu struct {
		sync.Mutex
		metrics []metric
	}
}

func (r *inMemoryRegistry) register(m metric) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mu.metrics = append(r.mu.metrics, m)
}

func (r *inMemoryRegistry) NewRate(name string) *Rate {
	m := newRate(name)
	r.register(m)
	return m
}

func (r *inMemoryRegistry) NewGauge(name string) *Gauge {
	g := &Gauge{namedMetric: namedMetric{name: name}}
	r.register(g)
	return g
}

func (r *inMemoryRegistry) NewLatency(name string, maxDuration time.Duration) *Histogram {
	h := newHistogram(name, maxDuration.Nanoseconds(), int(AGGREGATION_SPAN/REPORT_INTERVAL))
	r.register(h)
	return h
}

func (r *inMemoryRegistry) NewHistogram(name string, maxValue int64) *Histogram {
	h := newHistogram(name, maxValue, int(AGGREGATION_SPAN/REPORT_INTERVAL))
	r.register(h)
	return h
}

func (r *inMemoryRegistry) NewText(name string, defaultValue ...string) *Text {
	m := newText(name, defaultValue...)
	r.register(m)
	return m
}

func (r *inMemoryRegistry) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var s string
	for _, m := range r.mu.metrics {
		s += m.String()
	}

	return s
}

func (r *inMemoryRegistry) ExportAll() map[string]exportedMetric {
	r.mu.Lock()
	defer r.mu.Unlock()

	all := make(map[string]exportedMetric)
	for _, m := range r.mu.metrics {
		all[m.Name()] = m.Export()
	}

	return all
}

func (r *inMemoryRegistry) report(logger log.BasicLogger) {
	for _, value := range r.ExportAll() {
		if logRow := value.LogRow(); logRow != nil {
			logger.Metric(logRow...)
		}
	}
}

func (r *inMemoryRegistry) PeriodicallyReport(ctx context.Context, logger log.BasicLogger) {
	synchronization.NewPeriodicalTrigger(ctx, REPORT_INTERVAL, logger, func() {
		r.report(logger)

		// We only rotate histograms because there is the only type of metric that we're currently rotating
		r.mu.Lock()
		defer r.mu.Unlock()
		for _, m := range r.mu.metrics {
			switch m.(type) {
			case *Histogram:
				m.(*Histogram).Rotate()
			}
		}
	}, func() {
		r.report(logger)
	})
}

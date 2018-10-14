package metric

import (
	"fmt"
	"sync"
	"time"
)

type Factory interface {
	NewLatency(name string, maxDuration time.Duration) *Histogram
	NewGauge(name string) *Gauge
	NewRate(name string) *Rate
}

type Registry interface {
	Factory
	String() string
	ExportAll() map[string]interface{}
}

type metric interface {
	fmt.Stringer
	Name() string
	Export() interface{}
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
	h := newHistogram(name, maxDuration.Nanoseconds())
	r.register(h)
	return h
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

func (r *inMemoryRegistry) ExportAll() map[string]interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()

	all := make(map[string]interface{})
	for _, m := range r.mu.metrics {
		all[m.Name()] = m.Export()
	}

	return all
}

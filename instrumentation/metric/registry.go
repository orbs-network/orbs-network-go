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
}

type metric interface {
	fmt.Stringer
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

func (r *inMemoryRegistry) NewRate(name string) *Rate {
	r.mu.Lock()
	defer r.mu.Unlock()
	m := newRate(name)
	r.mu.metrics = append(r.mu.metrics, m)
	return m
}

func (r *inMemoryRegistry) NewGauge(name string) *Gauge {
	r.mu.Lock()
	defer r.mu.Unlock()
	g := &Gauge{name: name}
	r.mu.metrics = append(r.mu.metrics, g)
	return g
}

func (r *inMemoryRegistry) NewLatency(name string, maxDuration time.Duration) *Histogram {
	r.mu.Lock()
	defer r.mu.Unlock()
	h := newHistogram(name, maxDuration.Nanoseconds() / 1000 / 1000)
	r.mu.metrics = append(r.mu.metrics, h)
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




package metric

import (
	"sync"
	"time"
)

type Registry interface {
	NewLatency(name string, maxDuration time.Duration) *Histogram
	String() string
}

func NewRegistry() Registry {
	return &inMemoryRegistry{}
}

type inMemoryRegistry struct {
	mu struct {
		sync.Mutex
		histograms []*Histogram
	}
}

func (r *inMemoryRegistry) NewLatency(name string, maxDuration time.Duration) *Histogram {
	r.mu.Lock()
	defer r.mu.Unlock()
	h := newHistogram(name, maxDuration.Nanoseconds() / 1000 / 1000)
	r.mu.histograms = append(r.mu.histograms, h)
	return h
}

func (r *inMemoryRegistry) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var s string
	for _, h := range r.mu.histograms {
		s += h.String()
	}

	return s
}




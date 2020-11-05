// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package metric

import (
	"context"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"sync"
	"time"
)

const ROTATE_INTERVAL = 30 * time.Second
const AGGREGATION_SPAN = 10 * time.Minute

type Factory interface {
	NewHistogram(name string, maxValue int64) *Histogram
	NewHistogramWithPrometheusName(name string, pName string, maxValue int64) *Histogram
	NewLatency(name string, maxDuration time.Duration) *Histogram
	NewLatencyWithPrometheusName(name string, pName string, maxDuration time.Duration) *Histogram
	NewGauge(name string) *Gauge
	NewGaugeWithValue(name string, value int64) *Gauge
	NewGaugeWithPrometheusName(name string, pName string) *Gauge
	NewRate(name string) *Rate
	NewText(name string, defaultValue ...string) *Text
}

type Registry interface {
	Factory
	WithVirtualChainId(id primitives.VirtualChainId) Registry
	WithNodeAddress(nodeAddress primitives.NodeAddress) Registry
	Remove(metric metric)
	Get(metricName string) metric
	PeriodicallyRotate(ctx context.Context, logger log.Logger) govnr.ShutdownWaiter
	ExportAllNested(log log.Logger) exportedMap
	ExportPrometheus() string
}

type metric interface {
	Name() string
	Export() interface{}
	Value() interface{}
	exportPrometheus(labelString string) string
}

type exportedMap map[string]interface{}

type inMemoryRegistry struct {
	vcid        primitives.VirtualChainId
	nodeAddress primitives.NodeAddress
	mu          struct {
		sync.RWMutex
		metrics map[string]metric
	}
}

func NewRegistry() *inMemoryRegistry {
	r := &inMemoryRegistry{}
	r.mu.metrics = make(map[string]metric)
	return r
}

func (r *inMemoryRegistry) WithVirtualChainId(id primitives.VirtualChainId) Registry {
	r.vcid = id
	return r
}

func (r *inMemoryRegistry) WithNodeAddress(nodeAddress primitives.NodeAddress) Registry {
	r.nodeAddress = nodeAddress
	return r
}

func (r *inMemoryRegistry) Get(metricName string) metric {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.mu.metrics[metricName]
}

// Only if the actual metric (the object/pointer) is the same remove metric
func (r *inMemoryRegistry) Remove(m metric) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m == nil {
		return
	}

	if mapMetric, isMetricInMap := r.mu.metrics[m.Name()]; isMetricInMap {
		if mapMetric == m {
			delete(r.mu.metrics, m.Name())
		}
	}
}

func (r *inMemoryRegistry) register(m metric) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, isMetricInMap := r.mu.metrics[m.Name()]; isMetricInMap {
		err := errors.Errorf("a metric with name %s is already registered", m.Name())
		panic(err)
	}

	r.mu.metrics[m.Name()] = m
}

func (r *inMemoryRegistry) NewRate(name string) *Rate {
	m := newRate(name, name)
	r.register(m)
	return m
}

func (r *inMemoryRegistry) NewGauge(name string) *Gauge {
	return r.NewGaugeWithPrometheusName(name, name)
}

func (r *inMemoryRegistry) NewGaugeWithPrometheusName(name string, pName string) *Gauge {
	g := newGauge(name, pName)
	r.register(g)
	return g
}

func (r *inMemoryRegistry) NewGaugeWithValue(name string, value int64) *Gauge {
	g := newGauge(name, name)
	g.Update(value)
	r.register(g)
	return g
}

func (r *inMemoryRegistry) NewLatency(name string, maxDuration time.Duration) *Histogram {
	return r.NewLatencyWithPrometheusName(name, name, maxDuration)
}

func (r *inMemoryRegistry) NewLatencyWithPrometheusName(name string, pName string, maxDuration time.Duration) *Histogram {
	h := newHistogram(name, pName, maxDuration.Nanoseconds(), int(AGGREGATION_SPAN/ROTATE_INTERVAL))
	r.register(h)
	return h
}

func (r *inMemoryRegistry) NewHistogram(name string, maxValue int64) *Histogram {
	return r.NewHistogramWithPrometheusName(name, name, maxValue)
}

func (r *inMemoryRegistry) NewHistogramWithPrometheusName(name string, pName string, maxValue int64) *Histogram {
	h := newHistogram(name, pName, maxValue, int(AGGREGATION_SPAN/ROTATE_INTERVAL))
	r.register(h)
	return h
}

func (r *inMemoryRegistry) NewText(name string, defaultValue ...string) *Text {
	m := newText(name, name, defaultValue...)
	r.register(m)
	return m
}

func (r *inMemoryRegistry) PeriodicallyRotate(ctx context.Context, logger log.Logger) govnr.ShutdownWaiter {
	return synchronization.NewPeriodicalTrigger(ctx, "Metric registry rotation trigger", synchronization.NewTimeTicker(ROTATE_INTERVAL), logger, func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		for _, m := range r.mu.metrics {
			switch m.(type) {
			case *Histogram: // only Histograms currently require rotating
				m.(*Histogram).Rotate()
			}
		}
	}, nil)
}

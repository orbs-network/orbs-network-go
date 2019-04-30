// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package metric

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
	"strconv"
	"strings"
	"sync"
	"time"
)

const ROTATE_INTERVAL = 30 * time.Second
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
	PeriodicallyRotate(ctx context.Context, logger log.Logger)
	ExportPrometheus() string
	WithVirtualChainId(id primitives.VirtualChainId) Registry
	WithNodeAddress(nodeAddress primitives.NodeAddress) Registry
}

type exportedMetric interface {
	LogRow() []*log.Field
	PrometheusRow() []*prometheusRow
	PrometheusType() string
	PrometheusName() string
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
	vcid        primitives.VirtualChainId
	nodeAddress primitives.NodeAddress
	mu          struct {
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
	h := newHistogram(name, maxDuration.Nanoseconds(), int(AGGREGATION_SPAN/ROTATE_INTERVAL))
	r.register(h)
	return h
}

func (r *inMemoryRegistry) NewHistogram(name string, maxValue int64) *Histogram {
	h := newHistogram(name, maxValue, int(AGGREGATION_SPAN/ROTATE_INTERVAL))
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

func (r *inMemoryRegistry) PeriodicallyRotate(ctx context.Context, logger log.Logger) {
	synchronization.NewPeriodicalTrigger(ctx, ROTATE_INTERVAL, logger, func() {
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

func (r *inMemoryRegistry) ExportPrometheus() string {
	metrics := r.ExportAll()

	var params []prometheusKeyValuePair
	if r.vcid > 0 {
		vcid := strconv.FormatUint(uint64(r.vcid), 10)
		params = append(params, prometheusKeyValuePair{"vcid", vcid})
	}

	if r.nodeAddress != nil {
		params = append(params, prometheusKeyValuePair{"node", r.nodeAddress.String()})
	}

	var rows []string
	for _, v := range metrics {
		if v.PrometheusType() != "" {
			rows = append(rows, fmt.Sprintf("# TYPE %s %s", v.PrometheusName(), v.PrometheusType()))

			for _, row := range v.PrometheusRow() {
				rows = append(rows, row.String(params...))
			}
		}
	}

	return strings.Join(rows, "\n")
}

func (r *inMemoryRegistry) WithVirtualChainId(id primitives.VirtualChainId) Registry {
	r.vcid = id
	return r
}

func (r *inMemoryRegistry) WithNodeAddress(nodeAddress primitives.NodeAddress) Registry {
	r.nodeAddress = nodeAddress
	return r
}

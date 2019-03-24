// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package metric

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"sync/atomic"
)

type Gauge struct {
	namedMetric
	value int64
}

type gaugeExport struct {
	Name  string
	Value int64
}

func (g *Gauge) Export() exportedMetric {
	return gaugeExport{
		g.name,
		g.value,
	}
}

func (g *Gauge) String() string {
	return fmt.Sprintf("metric %s: %d\n", g.name, g.value)
}

func (g *Gauge) Inc() {
	g.Add(1)
}

func (g *Gauge) Add(i int64) {
	atomic.AddInt64(&g.value, i)
}

func (g *Gauge) AddUint32(i uint32) {
	g.Add(int64(i))
}

func (g *Gauge) Dec() {
	g.Add(-1)
}

func (g *Gauge) SubUint32(size uint32) {
	g.Add(-int64(size))
}

func (g *Gauge) Update(i int64) {
	atomic.StoreInt64(&g.value, i)
}

func (g *Gauge) UpdateUInt32(i int32) {
	atomic.StoreInt64(&g.value, int64(i))
}

func (g *Gauge) Value() int64 {
	return g.value
}

func (g gaugeExport) LogRow() []*log.Field {
	return []*log.Field{
		log.String("metric", g.Name),
		log.String("metric-type", "gauge"),
		log.Int64("gauge", g.Value),
	}
}

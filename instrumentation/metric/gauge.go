// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package metric

import (
	"sync/atomic"
)

type Gauge struct {
	name  string
	pName string
	value int64
}

func newGauge(name string, pName string) *Gauge {
	return &Gauge{name: name, pName: prometheusName(pName)}
}

func (g *Gauge) Name() string {
	return g.name
}

func (g *Gauge) Value() interface{} {
	return g.IntValue()
}

func (g *Gauge) Export() interface{} {
	return g.Value()
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

func (g *Gauge) IntValue() int64 {
	return atomic.LoadInt64(&g.value)
}

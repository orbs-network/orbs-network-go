package metric

import (
	"fmt"
	"sync/atomic"
)

type Gauge struct {
	value int64
	name  string
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

func (g *Gauge) Value() int64 {
	return g.value
}

